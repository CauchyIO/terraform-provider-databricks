package catalog

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/terraform-provider-databricks/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type SchemaInfo struct {
	Name                         string                       `json:"name" tf:"force_new"`
	CatalogName                  string                       `json:"catalog_name" tf:"force_new"`
	StorageRoot                  string                       `json:"storage_root,omitempty" tf:"force_new"`
	Comment                      string                       `json:"comment,omitempty"`
	Properties                   map[string]string            `json:"properties,omitempty"`
	EnablePredictiveOptimization string                       `json:"enable_predictive_optimization,omitempty" tf:"computed"`
	Owner                        string                       `json:"owner,omitempty" tf:"computed"`
	MetastoreID                  string                       `json:"metastore_id,omitempty" tf:"computed"`
	FullName                     string                       `json:"full_name,omitempty" tf:"computed"`
	SchemaID                     string                       `json:"schema_id" tf:"computed"`
	SecretStorage                *catalog.SchemaSecretStorage `json:"secret_storage,omitempty" tf:"max_items:1"`
	common.Namespace
}

func unboundSecretStorage() *catalog.SchemaSecretStorage {
	return &catalog.SchemaSecretStorage{
		StorageType: catalog.SchemaSecretStorageTypeStorageTypeUnspecified,
	}
}

// getSchema fetches a schema. Unity Catalog only returns secret_storage when it
// is asked for, so it is requested only when the resource manages a binding.
func getSchema(ctx context.Context, w *databricks.WorkspaceClient, fullName string, withSecretStorage bool) (*catalog.SchemaInfo, error) {
	if !withSecretStorage {
		return w.Schemas.GetByFullName(ctx, fullName)
	}
	return w.Schemas.Get(ctx, catalog.GetSchemaRequest{
		FullName:             fullName,
		IncludeSecretStorage: true,
	})
}

// flattenSecretStorage is the state form of the binding Unity Catalog reports.
// A schema without a binding is recorded as STORAGE_TYPE_UNSPECIFIED, and both
// fields are always written: StructToData leaves fields that come back empty
// untouched, which would keep the binding of a schema that was unbound outside
// of Terraform in state and hide the drift.
func flattenSecretStorage(storage *catalog.SchemaSecretStorage) []any {
	flat := map[string]any{
		"storage_type":    string(catalog.SchemaSecretStorageTypeStorageTypeUnspecified),
		"connection_name": "",
	}
	if storage != nil {
		flat["connection_name"] = storage.ConnectionName
		if storage.StorageType != "" {
			flat["storage_type"] = string(storage.StorageType)
		}
	}
	return []any{flat}
}

// secretStorageChange returns what an update has to send for secret_storage:
// nothing when it did not change, the configured binding when it did, and an
// explicit STORAGE_TYPE_UNSPECIFIED when the block was removed, because Unity
// Catalog keeps the binding when the field is merely absent from the request.
func secretStorageChange(d *schema.ResourceData, configured *catalog.SchemaSecretStorage) *catalog.SchemaSecretStorage {
	if !d.HasChange("secret_storage") {
		return nil
	}
	if configured == nil {
		return unboundSecretStorage()
	}
	return configured
}

// deleteSchema deletes the schema. Unity Catalog refuses to delete a schema that
// is bound to external secret storage, even with force, so such a schema is
// unbound first.
func deleteSchema(ctx context.Context, w *databricks.WorkspaceClient, d *schema.ResourceData) error {
	name := d.Id()
	request := catalog.DeleteSchemaRequest{FullName: name, Force: d.Get("force_destroy").(bool)}
	storageType := d.Get("secret_storage.0.storage_type").(string)
	if storageType != string(catalog.SchemaSecretStorageTypeStorageTypeExternal) {
		return w.Schemas.Delete(ctx, request)
	}
	_, err := w.Schemas.Update(ctx, catalog.UpdateSchema{
		FullName:      name,
		SecretStorage: unboundSecretStorage(),
	})
	if err != nil {
		return err
	}
	err = w.Schemas.Delete(ctx, request)
	if err != nil {
		return fmt.Errorf("schema %s was unbound from secret storage connection %s but could not be deleted: %w",
			name, d.Get("secret_storage.0.connection_name"), err)
	}
	return nil
}

func ResourceSchema() common.Resource {
	s := common.StructToSchema(SchemaInfo{},
		func(s map[string]*schema.Schema) map[string]*schema.Schema {
			delete(s, "full_name")
			s["force_destroy"] = &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			}
			common.CustomizeSchemaPath(s, "storage_root").SetCustomSuppressDiff(ucDirectoryPathSlashOnlySuppressDiff)
			s["storage_root"].DiffSuppressFunc = ucDirectoryPathSlashOnlySuppressDiff
			common.CustomizeSchemaPath(s, "name").SetCustomSuppressDiff(common.EqualFoldDiffSuppress)
			common.CustomizeSchemaPath(s, "catalog_name").SetCustomSuppressDiff(common.EqualFoldDiffSuppress)
			common.CustomizeSchemaPath(s, "enable_predictive_optimization").SetValidateFunc(
				validation.StringInSlice([]string{"DISABLE", "ENABLE", "INHERIT"}, false),
			)
			common.CustomizeSchemaPath(s, "secret_storage", "storage_type").SetRequired().SetValidateFunc(
				validation.StringInSlice([]string{"STORAGE_TYPE_EXTERNAL", "STORAGE_TYPE_UNSPECIFIED"}, false),
			)
			common.NamespaceCustomizeSchemaMap(s)
			return s
		})
	return common.Resource{
		Schema:        s,
		CustomizeDiff: common.NamespaceCustomizeDiffNoForceNew,
		Create: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			w, err := c.WorkspaceClientUnifiedProvider(ctx, d)
			if err != nil {
				return err
			}
			err = validateMetastoreId(ctx, w, d.Get("metastore_id").(string))
			if err != nil {
				return err
			}
			var createSchemaRequest catalog.CreateSchema
			common.DataToStructPointer(d, s, &createSchemaRequest)
			schema, err := w.Schemas.Create(ctx, createSchemaRequest)
			if err != nil {
				return err
			}
			d.SetId(schema.FullName)

			// Update owner or predictive optimization if it is provided
			updateRequired := false
			for _, key := range []string{"owner", "enable_predictive_optimization"} {
				if d.Get(key) != "" {
					updateRequired = true
					break
				}
			}
			if !updateRequired {
				return nil
			}

			var updateSchemaRequest catalog.UpdateSchema
			common.DataToStructPointer(d, s, &updateSchemaRequest)
			updateSchemaRequest.FullName = d.Id()
			// The binding was already sent on create.
			updateSchemaRequest.SecretStorage = nil
			_, err = w.Schemas.Update(ctx, updateSchemaRequest)
			if err != nil {
				return err
			}
			return nil
		},
		Read: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			w, err := c.WorkspaceClientUnifiedProvider(ctx, d)
			if err != nil {
				return err
			}
			_, managesSecretStorage := d.GetOk("secret_storage")
			schema, err := getSchema(ctx, w, d.Id(), managesSecretStorage)
			if err != nil {
				return err
			}
			err = common.StructToData(schema, s, d)
			if err != nil || !managesSecretStorage {
				return err
			}
			return d.Set("secret_storage", flattenSecretStorage(schema.SecretStorage))
		},
		Update: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			w, err := c.WorkspaceClientUnifiedProvider(ctx, d)
			if err != nil {
				return err
			}
			err = validateMetastoreId(ctx, w, d.Get("metastore_id").(string))
			if err != nil {
				return err
			}
			if !d.HasChangeExcept("force_destroy") {
				return nil
			}
			var updateSchemaRequest catalog.UpdateSchema
			common.DataToStructPointer(d, s, &updateSchemaRequest)
			updateSchemaRequest.FullName = d.Id()
			updateSchemaRequest.SecretStorage = secretStorageChange(d, updateSchemaRequest.SecretStorage)

			if d.HasChange("owner") {
				_, err := w.Schemas.Update(ctx, catalog.UpdateSchema{
					FullName: updateSchemaRequest.FullName,
					Owner:    updateSchemaRequest.Owner,
				})
				if err != nil {
					return err
				}
			}

			if !d.HasChangeExcept("owner") {
				return nil
			}

			if d.HasChange("comment") && updateSchemaRequest.Comment == "" {
				updateSchemaRequest.ForceSendFields = append(updateSchemaRequest.ForceSendFields, "Comment")
			}

			updateSchemaRequest.Owner = ""
			schema, err := w.Schemas.Update(ctx, updateSchemaRequest)
			if err != nil {
				if d.HasChange("owner") {
					// Rollback
					old, new := d.GetChange("owner")
					_, rollbackErr := w.Schemas.Update(ctx, catalog.UpdateSchema{
						FullName: updateSchemaRequest.FullName,
						Owner:    old.(string),
					})
					if rollbackErr != nil {
						return common.OwnerRollbackError(err, rollbackErr, old.(string), new.(string))
					}
				}
				return err
			}
			// We need to update the resource Id because Name is updatable and FullName consists of Name,
			// So if we don't update the field then the requests would be made to old FullName which doesn't exists.
			d.SetId(schema.FullName)
			return nil
		},
		Delete: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			w, err := c.WorkspaceClientUnifiedProvider(ctx, d)
			if err != nil {
				return err
			}
			err = validateMetastoreId(ctx, w, d.Get("metastore_id").(string))
			if err != nil {
				return err
			}
			return deleteSchema(ctx, w, d)
		},
	}
}
