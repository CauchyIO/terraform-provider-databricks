package catalog

import (
	"errors"
	"fmt"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/terraform-provider-databricks/qa"
	"github.com/stretchr/testify/mock"
)

func TestSchemaCornerCases(t *testing.T) {
	qa.ResourceCornerCases(t, ResourceSchema())
}

func TestCreateSchema(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			e := w.GetMockSchemasAPI().EXPECT()
			e.Create(mock.Anything, catalog.CreateSchema{
				Name:        "a",
				CatalogName: "b",
				Comment:     "c",
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "c",
				Owner:    "testers",
			}, nil)
			e.GetByFullName(mock.Anything, "b.a").Return(&catalog.SchemaInfo{
				MetastoreId: "d",
				Comment:     "c",
				Owner:       "testers",
			}, nil)
		},
		Resource: ResourceSchema(),
		Create:   true,
		HCL: `
		name = "a"
		catalog_name = "b"
		comment = "c"
		`,
	}.ApplyNoError(t)
}

func TestCreateSchemaWithOwner(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			e := w.GetMockSchemasAPI().EXPECT()
			e.Create(mock.Anything, catalog.CreateSchema{
				Name:        "a",
				CatalogName: "b",
				Comment:     "c",
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "c",
				Owner:    "testers",
			}, nil)
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName:                     "b.a",
				Comment:                      "c",
				Owner:                        "administrators",
				EnablePredictiveOptimization: "ENABLE",
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "c",
				Owner:    "administrators",
			}, nil)
			e.GetByFullName(mock.Anything, "b.a").Return(&catalog.SchemaInfo{
				MetastoreId:                  "d",
				Comment:                      "c",
				Owner:                        "administrators",
				EnablePredictiveOptimization: "ENABLE",
			}, nil)
		},
		Resource: ResourceSchema(),
		Create:   true,
		HCL: `
		name = "a"
		catalog_name = "b"
		comment = "c"
		owner = "administrators"
		enable_predictive_optimization = "ENABLE"
		`,
	}.ApplyNoError(t)
}

func TestUpdateSchema(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			w.GetMockMetastoresAPI().EXPECT().Current(mock.Anything).Return(&catalog.MetastoreAssignment{
				MetastoreId: "d",
			}, nil)
			e := w.GetMockSchemasAPI().EXPECT()
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName: "b.a",
				Owner:    "administrators",
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "c",
				Owner:    "administrators",
			}, nil)
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName: "b.a",
				Comment:  "d",
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "d",
				Owner:    "administrators",
			}, nil)
			e.GetByFullName(mock.Anything, "b.a").Return(&catalog.SchemaInfo{
				Name:        "a",
				CatalogName: "b",
				MetastoreId: "d",
				Comment:     "d",
				Owner:       "administrators",
			}, nil)
		},
		Resource: ResourceSchema(),
		Update:   true,
		ID:       "b.a",
		InstanceState: map[string]string{
			"metastore_id": "d",
			"name":         "a",
			"catalog_name": "b",
			"comment":      "c",
		},
		HCL: `
		name = "a"
		catalog_name = "b"
		comment = "d"
		owner = "administrators"
		`,
	}.ApplyNoError(t)
}

func TestUpdateSchemaSetEmptyComment(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			w.GetMockMetastoresAPI().EXPECT().Current(mock.Anything).Return(&catalog.MetastoreAssignment{
				MetastoreId: "d",
			}, nil)
			e := w.GetMockSchemasAPI().EXPECT()
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName:        "b.a",
				Comment:         "",
				ForceSendFields: []string{"Comment"},
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Owner:    "administrators",
			}, nil)
			e.GetByFullName(mock.Anything, "b.a").Return(&catalog.SchemaInfo{
				Name:        "a",
				CatalogName: "b",
				MetastoreId: "d",
				Owner:       "administrators",
			}, nil)
		},
		Resource: ResourceSchema(),
		Update:   true,
		ID:       "b.a",
		InstanceState: map[string]string{
			"metastore_id": "d",
			"name":         "a",
			"catalog_name": "b",
			"comment":      "c",
		},
		HCL: `
		name = "a"
		catalog_name = "b"
		`,
	}.ApplyAndExpectData(t, map[string]any{
		"comment": "",
	})
}

func TestUpdateSchemaChangeForceDestroy(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			w.GetMockMetastoresAPI().EXPECT().Current(mock.Anything).Return(&catalog.MetastoreAssignment{
				MetastoreId: "d",
			}, nil)
			e := w.GetMockSchemasAPI().EXPECT()
			e.GetByFullName(mock.Anything, "b.a").Return(&catalog.SchemaInfo{
				Name:        "a",
				CatalogName: "b",
				MetastoreId: "d",
				Owner:       "administrators",
			}, nil)
		},
		Resource: ResourceSchema(),
		Update:   true,
		ID:       "b.a",
		InstanceState: map[string]string{
			"metastore_id":  "d",
			"name":          "a",
			"catalog_name":  "b",
			"force_destroy": "true",
		},
		HCL: `
		name = "a"
		catalog_name = "b"
		force_destroy = false
		`,
	}.ApplyNoError(t)
}

func TestUpdateSchemaOwnerWithOtherFields(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			e := w.GetMockSchemasAPI().EXPECT()
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName: "b.a",
				Owner:    "administrators",
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "c",
				Owner:    "administrators",
			}, nil)
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName: "b.a",
				Comment:  "d",
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "d",
				Owner:    "administrators",
			}, nil)
			e.GetByFullName(mock.Anything, "b.a").Return(&catalog.SchemaInfo{
				Name:        "a",
				CatalogName: "b",
				MetastoreId: "d",
				Comment:     "d",
				Owner:       "administrators",
			}, nil)
		},
		Resource: ResourceSchema(),
		Update:   true,
		ID:       "b.a",
		InstanceState: map[string]string{
			"name":         "a",
			"catalog_name": "b",
			"comment":      "c",
			"owner":        "testOwner",
		},
		HCL: `
		name = "a"
		catalog_name = "b"
		comment = "d"
		owner = "administrators"
		`,
	}.ApplyNoError(t)
}

func TestUpdateSchemaRollback(t *testing.T) {
	_, err := qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			e := w.GetMockSchemasAPI().EXPECT()
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName: "b.a",
				Owner:    "administrators",
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "c",
				Owner:    "administrators",
			}, nil)
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName: "b.a",
				Comment:  "d",
			}).Return(nil, errors.New("Something unexpected happened"))
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName: "b.a",
				Owner:    "testOwner",
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "c",
				Owner:    "testOwner",
			}, nil)
		},
		Resource: ResourceSchema(),
		Update:   true,
		ID:       "b.a",
		InstanceState: map[string]string{
			"name":         "a",
			"catalog_name": "b",
			"comment":      "c",
			"owner":        "testOwner",
		},
		HCL: `
		name = "a"
		catalog_name = "b"
		comment = "d"
		owner = "administrators"
		`,
	}.Apply(t)
	qa.AssertErrorStartsWith(t, err, "Something unexpected happened")
}

func TestUpdateSchemaRollback_Error(t *testing.T) {
	serverErrMessage := "Something unexpected happened"
	rollbackErrMessage := "Internal error happened"
	_, err := qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			e := w.GetMockSchemasAPI().EXPECT()
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName: "b.a",
				Owner:    "administrators",
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "c",
				Owner:    "administrators",
			}, nil)
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName: "b.a",
				Comment:  "d",
			}).Return(nil, errors.New(serverErrMessage))
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName: "b.a",
				Owner:    "testOwner",
			}).Return(nil, errors.New(rollbackErrMessage))
		},
		Resource: ResourceSchema(),
		Update:   true,
		ID:       "b.a",
		InstanceState: map[string]string{
			"name":         "a",
			"catalog_name": "b",
			"comment":      "c",
			"owner":        "testOwner",
		},
		HCL: `
		name = "a"
		catalog_name = "b"
		comment = "d"
		owner = "administrators"
		`,
	}.Apply(t)
	errOccurred := fmt.Sprintf("%s. Owner rollback also failed: %s", serverErrMessage, rollbackErrMessage)
	qa.AssertErrorStartsWith(t, err, errOccurred)
}

func TestUpdateSchemaForceNew(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			e := w.GetMockSchemasAPI().EXPECT()
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName: "b.a",
				Owner:    "administrators",
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "c",
				Owner:    "administrators",
			}, nil)
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName: "b.a",
				Comment:  "c",
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "c",
				Owner:    "administrators",
			}, nil)
			e.GetByFullName(mock.Anything, "b.a").Return(&catalog.SchemaInfo{
				Name:        "a",
				MetastoreId: "d",
				Comment:     "c",
				Owner:       "administrators",
			}, nil)
		},
		RequiresNew: true,
		Resource:    ResourceSchema(),
		Update:      true,
		ID:          "b.a",
		InstanceState: map[string]string{
			"metastore_id": "d",
			"name":         "a",
			"catalog_name": "b",
			"comment":      "c",
		},
		HCL: `
		name = "a"
		catalog_name = "x"
		comment = "c"
		owner = "administrators"
		`,
	}.ApplyNoError(t)
}

func TestDeleteSchema(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			w.GetMockSchemasAPI().EXPECT().Delete(mock.Anything, catalog.DeleteSchemaRequest{FullName: "b.a"}).Return(nil)
		},
		Resource: ResourceSchema(),
		Delete:   true,
		ID:       "b.a",
		HCL: `
		name = "a"
		catalog_name = "b"
		comment = "c"
		owner = "administrators"
		`,
	}.ApplyNoError(t)
}

func TestForceDeleteSchema(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			w.GetMockSchemasAPI().EXPECT().Delete(mock.Anything, catalog.DeleteSchemaRequest{FullName: "b.a", Force: true}).Return(nil)
		},
		Resource: ResourceSchema(),
		Delete:   true,
		ID:       "b.a",
		HCL: `
		name = "a"
		catalog_name = "b"
		comment = "c"
		owner = "administrators"
		force_destroy = true
		`,
	}.ApplyNoError(t)
}

const secretStorageHCL = `
		secret_storage {
			storage_type = "STORAGE_TYPE_EXTERNAL"
			connection_name = "kv_uc_secrets_demo"
		}
		`

func testBoundSecretStorage() *catalog.SchemaSecretStorage {
	return &catalog.SchemaSecretStorage{
		ConnectionName: "kv_uc_secrets_demo",
		StorageType:    catalog.SchemaSecretStorageTypeStorageTypeExternal,
	}
}

func testUnboundSecretStorage() *catalog.SchemaSecretStorage {
	return &catalog.SchemaSecretStorage{
		StorageType: catalog.SchemaSecretStorageTypeStorageTypeUnspecified,
	}
}

// boundSchemaState is the prior state of schema `b.a` bound to a connection.
func boundSchemaState() map[string]string {
	return map[string]string{
		"metastore_id":                     "d",
		"name":                             "a",
		"catalog_name":                     "b",
		"comment":                          "c",
		"secret_storage.#":                 "1",
		"secret_storage.0.storage_type":    "STORAGE_TYPE_EXTERNAL",
		"secret_storage.0.connection_name": "kv_uc_secrets_demo",
	}
}

func TestCreateSchemaWithSecretStorage(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			e := w.GetMockSchemasAPI().EXPECT()
			e.Create(mock.Anything, catalog.CreateSchema{
				Name:          "a",
				CatalogName:   "b",
				Comment:       "c",
				SecretStorage: testBoundSecretStorage(),
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "c",
				Owner:    "testers",
			}, nil)
			e.Get(mock.Anything, catalog.GetSchemaRequest{
				FullName:             "b.a",
				IncludeSecretStorage: true,
			}).Return(&catalog.SchemaInfo{
				MetastoreId:   "d",
				Comment:       "c",
				Owner:         "testers",
				SecretStorage: testBoundSecretStorage(),
			}, nil)
		},
		Resource: ResourceSchema(),
		Create:   true,
		HCL: `
		name = "a"
		catalog_name = "b"
		comment = "c"
		` + secretStorageHCL,
	}.ApplyAndExpectData(t, map[string]any{
		"id":                               "b.a",
		"secret_storage.#":                 1,
		"secret_storage.0.storage_type":    "STORAGE_TYPE_EXTERNAL",
		"secret_storage.0.connection_name": "kv_uc_secrets_demo",
	})
}

// The binding is sent on create, so the follow-up update that sets the owner
// must not send it a second time.
func TestCreateSchemaWithSecretStorageAndOwner(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			e := w.GetMockSchemasAPI().EXPECT()
			e.Create(mock.Anything, catalog.CreateSchema{
				Name:          "a",
				CatalogName:   "b",
				SecretStorage: testBoundSecretStorage(),
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Owner:    "testers",
			}, nil)
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName: "b.a",
				Owner:    "administrators",
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Owner:    "administrators",
			}, nil)
			e.Get(mock.Anything, catalog.GetSchemaRequest{
				FullName:             "b.a",
				IncludeSecretStorage: true,
			}).Return(&catalog.SchemaInfo{
				MetastoreId:   "d",
				Owner:         "administrators",
				SecretStorage: testBoundSecretStorage(),
			}, nil)
		},
		Resource: ResourceSchema(),
		Create:   true,
		HCL: `
		name = "a"
		catalog_name = "b"
		owner = "administrators"
		` + secretStorageHCL,
	}.ApplyNoError(t)
}

func TestCreateSchemaWithInvalidSecretStorageType(t *testing.T) {
	qa.ResourceFixture{
		Resource: ResourceSchema(),
		Create:   true,
		HCL: `
		name = "a"
		catalog_name = "b"
		secret_storage {
			storage_type = "STORAGE_TYPE_INTERNAL"
		}
		`,
	}.ExpectError(t, "invalid config supplied. [secret_storage.#.storage_type] expected secret_storage.0.storage_type to be one of [STORAGE_TYPE_EXTERNAL STORAGE_TYPE_UNSPECIFIED], got STORAGE_TYPE_INTERNAL")
}

func TestUpdateSchemaBindSecretStorage(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			w.GetMockMetastoresAPI().EXPECT().Current(mock.Anything).Return(&catalog.MetastoreAssignment{
				MetastoreId: "d",
			}, nil)
			e := w.GetMockSchemasAPI().EXPECT()
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName:      "b.a",
				Comment:       "c",
				SecretStorage: testBoundSecretStorage(),
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "c",
			}, nil)
			e.Get(mock.Anything, catalog.GetSchemaRequest{
				FullName:             "b.a",
				IncludeSecretStorage: true,
			}).Return(&catalog.SchemaInfo{
				Name:          "a",
				CatalogName:   "b",
				MetastoreId:   "d",
				Comment:       "c",
				SecretStorage: testBoundSecretStorage(),
			}, nil)
		},
		Resource: ResourceSchema(),
		Update:   true,
		ID:       "b.a",
		InstanceState: map[string]string{
			"metastore_id": "d",
			"name":         "a",
			"catalog_name": "b",
			"comment":      "c",
		},
		HCL: `
		name = "a"
		catalog_name = "b"
		comment = "c"
		` + secretStorageHCL,
	}.ApplyAndExpectData(t, map[string]any{
		"secret_storage.0.storage_type":    "STORAGE_TYPE_EXTERNAL",
		"secret_storage.0.connection_name": "kv_uc_secrets_demo",
	})
}

// Removing the secret_storage block has to unbind the schema explicitly: Unity
// Catalog keeps the binding when the field is merely absent from the update.
func TestUpdateSchemaUnbindSecretStorage(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			w.GetMockMetastoresAPI().EXPECT().Current(mock.Anything).Return(&catalog.MetastoreAssignment{
				MetastoreId: "d",
			}, nil)
			e := w.GetMockSchemasAPI().EXPECT()
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName:      "b.a",
				Comment:       "c",
				SecretStorage: testUnboundSecretStorage(),
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "c",
			}, nil)
			e.GetByFullName(mock.Anything, "b.a").Return(&catalog.SchemaInfo{
				Name:        "a",
				CatalogName: "b",
				MetastoreId: "d",
				Comment:     "c",
			}, nil)
		},
		Resource:      ResourceSchema(),
		Update:        true,
		ID:            "b.a",
		InstanceState: boundSchemaState(),
		HCL: `
		name = "a"
		catalog_name = "b"
		comment = "c"
		`,
	}.ApplyAndExpectData(t, map[string]any{
		"secret_storage.#": 0,
	})
}

// An update that does not touch secret_storage must leave the binding out of
// the request instead of re-sending it.
func TestUpdateSchemaWithUnchangedSecretStorage(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			w.GetMockMetastoresAPI().EXPECT().Current(mock.Anything).Return(&catalog.MetastoreAssignment{
				MetastoreId: "d",
			}, nil)
			e := w.GetMockSchemasAPI().EXPECT()
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName: "b.a",
				Comment:  "new comment",
			}).Return(&catalog.SchemaInfo{
				FullName: "b.a",
				Comment:  "new comment",
			}, nil)
			e.Get(mock.Anything, catalog.GetSchemaRequest{
				FullName:             "b.a",
				IncludeSecretStorage: true,
			}).Return(&catalog.SchemaInfo{
				Name:          "a",
				CatalogName:   "b",
				MetastoreId:   "d",
				Comment:       "new comment",
				SecretStorage: testBoundSecretStorage(),
			}, nil)
		},
		Resource:      ResourceSchema(),
		Update:        true,
		ID:            "b.a",
		InstanceState: boundSchemaState(),
		HCL: `
		name = "a"
		catalog_name = "b"
		comment = "new comment"
		` + secretStorageHCL,
	}.ApplyNoError(t)
}

// A schema that was unbound outside of Terraform comes back without
// secret_storage. That has to show up in state, otherwise the binding is never
// restored.
func TestReadSchemaSecretStorageUnboundOutsideTerraform(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			w.GetMockSchemasAPI().EXPECT().Get(mock.Anything, catalog.GetSchemaRequest{
				FullName:             "b.a",
				IncludeSecretStorage: true,
			}).Return(&catalog.SchemaInfo{
				Name:        "a",
				CatalogName: "b",
				MetastoreId: "d",
			}, nil)
		},
		Resource: ResourceSchema(),
		Read:     true,
		ID:       "b.a",
		HCL: `
		name = "a"
		catalog_name = "b"
		` + secretStorageHCL,
	}.ApplyAndExpectData(t, map[string]any{
		"secret_storage.#":                 1,
		"secret_storage.0.storage_type":    "STORAGE_TYPE_UNSPECIFIED",
		"secret_storage.0.connection_name": "",
	})
}

// Unity Catalog refuses to delete a schema bound to external secret storage,
// even with force, so the schema has to be unbound before it is deleted.
func TestDeleteSchemaWithSecretStorageUnbindsFirst(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			e := w.GetMockSchemasAPI().EXPECT()
			unbind := e.Update(mock.Anything, catalog.UpdateSchema{
				FullName:      "b.a",
				SecretStorage: testUnboundSecretStorage(),
			}).Return(&catalog.SchemaInfo{FullName: "b.a"}, nil)
			del := e.Delete(mock.Anything, catalog.DeleteSchemaRequest{
				FullName: "b.a",
				Force:    true,
			}).Return(nil)
			mock.InOrder(unbind.Call, del.Call)
		},
		Resource: ResourceSchema(),
		Delete:   true,
		ID:       "b.a",
		HCL: `
		name = "a"
		catalog_name = "b"
		force_destroy = true
		` + secretStorageHCL,
	}.ApplyNoError(t)
}

func TestDeleteSchemaWithUnspecifiedSecretStorageSkipsUnbind(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			w.GetMockSchemasAPI().EXPECT().Delete(mock.Anything, catalog.DeleteSchemaRequest{FullName: "b.a"}).Return(nil)
		},
		Resource: ResourceSchema(),
		Delete:   true,
		ID:       "b.a",
		HCL: `
		name = "a"
		catalog_name = "b"
		secret_storage {
			storage_type = "STORAGE_TYPE_UNSPECIFIED"
		}
		`,
	}.ApplyNoError(t)
}

// When the unbind fails the schema is still bound, so the delete is not
// attempted and the unbind error is what the user sees.
func TestDeleteSchemaWithSecretStorageUnbindFails(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			w.GetMockSchemasAPI().EXPECT().Update(mock.Anything, catalog.UpdateSchema{
				FullName:      "b.a",
				SecretStorage: testUnboundSecretStorage(),
			}).Return(nil, errors.New("permission denied"))
		},
		Resource: ResourceSchema(),
		Delete:   true,
		ID:       "b.a",
		HCL: `
		name = "a"
		catalog_name = "b"
		` + secretStorageHCL,
	}.ExpectError(t, "permission denied")
}

// When the delete fails after a successful unbind, the schema is left unbound.
// The error has to say so, because the remote state no longer matches what
// Terraform recorded.
func TestDeleteSchemaWithSecretStorageDeleteFailsAfterUnbind(t *testing.T) {
	qa.ResourceFixture{
		MockWorkspaceClientFunc: func(w *mocks.MockWorkspaceClient) {
			e := w.GetMockSchemasAPI().EXPECT()
			e.Update(mock.Anything, catalog.UpdateSchema{
				FullName:      "b.a",
				SecretStorage: testUnboundSecretStorage(),
			}).Return(&catalog.SchemaInfo{FullName: "b.a"}, nil)
			e.Delete(mock.Anything, catalog.DeleteSchemaRequest{FullName: "b.a"}).Return(errors.New("schema is not empty"))
		},
		Resource: ResourceSchema(),
		Delete:   true,
		ID:       "b.a",
		HCL: `
		name = "a"
		catalog_name = "b"
		` + secretStorageHCL,
	}.ExpectError(t, "schema b.a was unbound from secret storage connection kv_uc_secrets_demo but could not be deleted: schema is not empty")
}
