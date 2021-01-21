package testtool

import (
	"cloudservices/common/base"
	"cloudservices/common/model"

	"context"
	"fmt"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
)

type whoCanDoIt struct {
	// Any tenant member can do it
	anybody bool

	// Any tenant's infra admin can do it
	anyInfraAdmin bool

	// Tenant's infra admin with assigned project can do it
	projectInfraAdmin bool

	// Tenant's user can do it
	anyUser bool

	// Tenant's user with assigned project can do it
	projectUser bool
}

type permissionMatrix struct {
	read   whoCanDoIt
	create whoCanDoIt
	update whoCanDoIt
	delete whoCanDoIt
}

func InfraProjectLevelObject() permissionMatrix {
	read := whoCanDoIt{
		anybody:           false,
		anyInfraAdmin:     true,
		projectInfraAdmin: true,
		anyUser:           false,
		projectUser:       true,
	}
	write := whoCanDoIt{
		anybody:           false,
		anyInfraAdmin:     false,
		projectInfraAdmin: true,
		anyUser:           false,
		projectUser:       false,
	}
	return permissionMatrix{read, write, write, write}
}

func ProjectLevelObject() permissionMatrix {
	perm := whoCanDoIt{
		anybody:           false,
		anyInfraAdmin:     false,
		projectInfraAdmin: true,
		anyUser:           false,
		projectUser:       true,
	}
	return permissionMatrix{perm, perm, perm, perm}
}

type TestAutomata struct {
	name        string
	permissions permissionMatrix

	tenantID         string
	projectID        string
	anotherProjectID string

	impostorTenantID  string
	impostorProjectID string

	selectors []func(t *testing.T, ctx context.Context, tenantId, projectId string) []string
	checker   func(t *testing.T, ctx context.Context, id string) error
	creator   func(t *testing.T, ctx context.Context, id, tenantId, projectId string) (string, error)
	updater   func(t *testing.T, ctx context.Context, id, tenantId, projectId string) error
	deleter   func(t *testing.T, ctx context.Context, id string) error
}

func APITestTool(name string) *TestAutomata {
	return &TestAutomata{name: name}
}

func (tb *TestAutomata) ForTenant(tenantID string, projectID string, anotherProjectID string) *TestAutomata {
	tb.tenantID = tenantID
	tb.projectID = projectID
	tb.anotherProjectID = anotherProjectID
	return tb
}

func (tb *TestAutomata) ForImpostor(tenantID string, projectID string) *TestAutomata {
	tb.impostorTenantID = tenantID
	tb.impostorProjectID = projectID
	return tb
}

func (tb *TestAutomata) PermissionsMatrix(permissions permissionMatrix) *TestAutomata {
	tb.permissions = permissions
	return tb
}

func (tb *TestAutomata) WithSelector(fn func(ctx context.Context, tenantId, projectId string) (interface{}, error)) *TestAutomata {
	if fn == nil {
		return tb
	}
	tb.selectors = append(tb.selectors, func(t *testing.T, ctx context.Context, tenantId, projectId string) []string {
		rsp, err := fn(ctx, tenantId, projectId)
		if err != nil {
			return []string{}
		}
		require.NotNil(t, rsp)
		return funk.Map(rsp, func(i interface{}) string {
			return i.(model.IdentifiableEntity).GetID()
		}).([]string)
	})
	return tb
}

func (tb *TestAutomata) WithChecker(fn func(ctx context.Context, id string) (interface{}, error)) *TestAutomata {
	tb.checker = func(t *testing.T, ctx context.Context, id string) error {
		res, err := fn(ctx, id)
		if err != nil {
			return err
		}
		require.NotNil(t, res)
		gotId := res.(model.IdentifiableEntity).GetID()
		require.Equal(t, id, gotId)
		return nil
	}
	return tb
}

func (tb *TestAutomata) WithCreator(fn func(ctx context.Context, id, tenantId, projectId string) (interface{}, error)) *TestAutomata {
	tb.creator = func(t *testing.T, ctx context.Context, id, tenantId, projectId string) (string, error) {
		rsp, err := fn(ctx, id, tenantId, projectId)
		if err != nil {
			return "", err
		}
		require.IsType(t, rsp, model.CreateDocumentResponseV2{})
		return rsp.(model.CreateDocumentResponseV2).ID, nil
	}
	return tb
}

func (tb *TestAutomata) WithUpdater(fn func(ctx context.Context, id, tenantId, projectId string) (interface{}, error)) *TestAutomata {
	tb.updater = func(t *testing.T, ctx context.Context, id, tenantId, projectId string) error {
		res, err := fn(ctx, id, tenantId, projectId)
		if err != nil {
			return err
		}
		require.NotNil(t, res)
		return nil
	}
	return tb
}

func (tb *TestAutomata) WithDeleter(fn func(ctx context.Context, id string) (interface{}, error)) *TestAutomata {
	tb.deleter = func(t *testing.T, ctx context.Context, id string) error {
		rsp, err := fn(ctx, id)
		if err != nil {
			return err
		}
		require.IsType(t, rsp, model.DeleteDocumentResponseV2{})
		require.Equal(t, rsp.(model.DeleteDocumentResponseV2).ID, id)
		return nil
	}
	return tb
}

func (tb *TestAutomata) shouldHaveProjectData(t *testing.T) {
	require.NotEmpty(t, tb.tenantID)
	require.NotEmpty(t, tb.projectID)
	require.NotEmpty(t, tb.anotherProjectID)

	require.NotEmpty(t, tb.impostorTenantID)
	require.NotEmpty(t, tb.impostorProjectID)
}

func (tb *TestAutomata) makeContexts(tenantID string, projectID string) (context.Context, context.Context) {
	projRoles := []model.ProjectRole{model.ProjectRole{
		ProjectID: projectID,
		Role:      model.ProjectRoleAdmin,
	}}
	adminContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"projects":    projRoles,
		},
	}
	userContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"projects": projRoles,
		},
	}
	ctx1 := context.WithValue(context.Background(), base.AuthContextKey, adminContext)
	ctx2 := context.WithValue(context.Background(), base.AuthContextKey, userContext)
	return ctx1, ctx2

}

func (tb *TestAutomata) adminContext() context.Context {
	adminContext, _ := tb.makeContexts(tb.tenantID, tb.projectID)
	return adminContext
}

func (tb *TestAutomata) checkWhoCanDoIt(t *testing.T, who whoCanDoIt, fn func(ctx context.Context) error) {
	tb.shouldHaveProjectData(t)

	adminContext, userContext := tb.makeContexts(tb.tenantID, tb.projectID)
	anotherAdminContext, anotherUserContext := tb.makeContexts(tb.tenantID, tb.anotherProjectID)
	impostorAdminContext, impostorUserContext := tb.makeContexts(tb.impostorTenantID, tb.impostorProjectID)

	require.Error(t, fn(impostorAdminContext))
	require.Error(t, fn(impostorUserContext))

	if who.anybody {
		require.NoError(t, fn(adminContext))
		require.NoError(t, fn(userContext))
		require.NoError(t, fn(anotherAdminContext))
		require.NoError(t, fn(anotherUserContext))
	} else {
		if who.anyInfraAdmin {
			require.NoError(t, fn(anotherAdminContext))
		} else {
			require.Error(t, fn(anotherAdminContext))
		}

		if who.projectInfraAdmin {
			require.NoError(t, fn(adminContext))
		} else {
			require.Error(t, fn(adminContext))
		}

		if who.anyUser {
			require.NoError(t, fn(anotherUserContext))
		} else {
			require.Error(t, fn(anotherUserContext))
		}

		if who.projectUser {
			require.NoError(t, fn(userContext))
		} else {
			require.Error(t, fn(userContext))
		}
	}
}

func (tb *TestAutomata) ReadRBACTest() func(*testing.T) {
	return func(t *testing.T) {
		require.NotNil(t, tb.creator)
		require.NotNil(t, tb.deleter)

		t.Logf("Starting READ test for '%v'", tb.name)
		adminContext := tb.adminContext()

		objectID, err := tb.creator(t, adminContext, base.GetUUID(), tb.tenantID, tb.projectID)
		require.NoError(t, err)

		tb.checkWhoCanDoIt(t, tb.permissions.read, func(ctx context.Context) error {
			return tb.checker(t, ctx, objectID)
		})

		t.Log("Do cleanup")
		require.NoError(t, tb.deleter(t, adminContext, objectID))
	}
}

func (tb *TestAutomata) SearchRBACTest() func(*testing.T) {
	return func(t *testing.T) {
		require.NotNil(t, tb.creator)
		require.NotNil(t, tb.deleter)
		require.NotEmpty(t, tb.selectors)
		tb.shouldHaveProjectData(t)

		adminContext, userContext := tb.makeContexts(tb.tenantID, tb.projectID)
		anotherAdminContext, anotherUserContext := tb.makeContexts(tb.tenantID, tb.anotherProjectID)
		impostorAdminContext, impostorUserContext := tb.makeContexts(tb.impostorTenantID, tb.impostorProjectID)

		checkEmpty := func() {
			for _, selector := range tb.selectors {
				require.NotNil(t, selector)

				require.Empty(t, selector(t, adminContext, tb.tenantID, tb.projectID), "should be empty for admin")
				require.Empty(t, selector(t, userContext, tb.tenantID, tb.projectID), "should be empty for user")

				require.Empty(t, selector(t, anotherAdminContext, tb.tenantID, tb.anotherProjectID), "should be empty for another admin")
				require.Empty(t, selector(t, anotherUserContext, tb.tenantID, tb.anotherProjectID), "should be empty for another user")

				require.Empty(t, selector(t, impostorAdminContext, tb.tenantID, tb.projectID), "should be empty for impostor")
				require.Empty(t, selector(t, impostorUserContext, tb.tenantID, tb.projectID), "should be empty for impostor user")
			}
		}

		checkEmpty()

		objectID, err := tb.creator(t, adminContext, base.GetUUID(), tb.tenantID, tb.projectID)
		require.NoError(t, err)

		anotherObjectID, err := tb.creator(t, anotherAdminContext, base.GetUUID(), tb.tenantID, tb.anotherProjectID)
		require.NoError(t, err)

		who := tb.permissions.read

		for count, selector := range tb.selectors {
			require.NotNil(t, selector)

			counter := func(ctx context.Context) int {
				found := selector(t, ctx, tb.tenantID, tb.projectID)
				return len(found)
			}

			t.Run(fmt.Sprintf("Starting SEARCH test for '%v' (%d)", tb.name, count), func(t *testing.T) {
				require.Equal(t, 0, counter(impostorAdminContext))
				require.Equal(t, 0, counter(impostorUserContext))
				if who.anybody {
					require.Equal(t, 2, counter(adminContext))
					require.Equal(t, 2, counter(userContext))
					require.Equal(t, 2, counter(anotherAdminContext))
					require.Equal(t, 2, counter(anotherUserContext))
				} else {
					if who.anyInfraAdmin {
						require.Equal(t, 2, counter(adminContext))
						require.Equal(t, 2, counter(anotherAdminContext))
					} else if who.projectInfraAdmin {
						require.Equal(t, 1, counter(adminContext))
					} else {
						require.Equal(t, 0, counter(adminContext))
						require.Equal(t, 0, counter(anotherAdminContext))
					}

					if who.anyUser {
						require.Equal(t, 2, counter(userContext))
						require.Equal(t, 2, counter(anotherUserContext))
					} else if who.projectUser {
						require.Equal(t, 1, counter(userContext))
					} else {
						require.Equal(t, 0, counter(userContext))
						require.Equal(t, 0, counter(anotherUserContext))
					}
				}
			})
		}

		t.Log("Do cleanup")
		require.NoError(t, tb.deleter(t, adminContext, objectID))
		require.NoError(t, tb.deleter(t, anotherAdminContext, anotherObjectID))

		checkEmpty()
	}
}

func (tb *TestAutomata) CreateRBACTest() func(*testing.T) {
	return func(t *testing.T) {
		require.NotNil(t, tb.creator)
		require.NotNil(t, tb.updater)
		require.NotNil(t, tb.deleter)

		tb.shouldHaveProjectData(t)

		t.Logf("Starting CREATE test for '%v'", tb.name)
		adminContext := tb.adminContext()

		tb.checkWhoCanDoIt(t, tb.permissions.delete, func(ctx context.Context) error {
			id, err := tb.creator(t, ctx, base.GetUUID(), tb.tenantID, tb.projectID)
			if err != nil {
				return err
			}

			err = tb.deleter(t, adminContext, id)
			require.NoError(t, err)

			return nil
		})
	}
}

func (tb *TestAutomata) UpdateRBACTest() func(*testing.T) {
	return func(t *testing.T) {
		require.NotNil(t, tb.creator)
		require.NotNil(t, tb.deleter)

		t.Logf("Starting UPDATE test for '%v'", tb.name)
		adminContext := tb.adminContext()

		objectID, err := tb.creator(t, adminContext, base.GetUUID(), tb.tenantID, tb.projectID)
		require.NoError(t, err)

		tb.checkWhoCanDoIt(t, tb.permissions.update, func(ctx context.Context) error {
			return tb.updater(t, ctx, objectID, tb.tenantID, tb.projectID)
		})

		t.Log("Do cleanup")
		require.NoError(t, tb.deleter(t, adminContext, objectID))
	}
}

func (tb *TestAutomata) DeleteRBACTest() func(*testing.T) {
	return func(t *testing.T) {
		require.NotNil(t, tb.creator)
		require.NotNil(t, tb.deleter)

		t.Logf("Starting DELETE test for '%v'", tb.name)
		adminContext := tb.adminContext()

		objectID, err := tb.creator(t, adminContext, base.GetUUID(), tb.tenantID, tb.projectID)
		require.NoError(t, err)

		tb.checkWhoCanDoIt(t, tb.permissions.delete, func(ctx context.Context) error {
			err := tb.deleter(t, ctx, objectID)
			if err != nil {
				return err
			}
			objectID, err = tb.creator(t, adminContext, base.GetUUID(), tb.tenantID, tb.projectID)
			return err
		})

		t.Log("Do cleanup")
		require.NoError(t, tb.deleter(t, adminContext, objectID))
	}
}

func (tb *TestAutomata) IdSanityTest() func(t *testing.T) {
	return func(t *testing.T) {
		require.NotNil(t, tb.creator)
		require.NotNil(t, tb.checker)

		require.NotEmpty(t, tb.tenantID)
		require.NotEmpty(t, tb.projectID)

		adminContext := tb.adminContext()

		for _, desc := range []struct {
			keep bool
			id   string
			name string
		}{
			{false, "", "Create with empty ID"},
			{false, funk.RandomString(37), "Create with long custom ID"},
			{false, funk.RandomString(7), "Create with short custom ID"},
			{true, base.GetUUID(), "Create with UUID ID"},
		} {
			t.Run(desc.name, func(t *testing.T) {
				invalidId := desc.id
				createdID, err := tb.creator(t, adminContext, invalidId, tb.tenantID, tb.projectID)
				require.NoError(t, err)

				if desc.keep {
					require.Equal(t, createdID, invalidId)
				} else {
					_, err = uuid.Parse(createdID)
					require.NoErrorf(t, err, "Created object ID is not UUID id=%s", createdID)
					require.NotEqual(t, createdID, invalidId)

					err = tb.checker(t, adminContext, invalidId)
					require.Errorf(t, err, "Should not be available via invalid id=%s", invalidId)
				}

				err = tb.checker(t, adminContext, createdID)
				require.NoErrorf(t, err, "Should be available via invalid id=%s", createdID)

				err = tb.deleter(t, adminContext, createdID)
				require.NoErrorf(t, err, "Failed to delete id=%s", createdID)
			})
		}
	}
}
