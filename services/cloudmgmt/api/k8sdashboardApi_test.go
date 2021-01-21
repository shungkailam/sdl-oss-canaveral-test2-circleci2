package api_test

import (
	"cloudservices/common/model"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	funk "github.com/thoas/go-funk"
)

func TestK8sDashboardApi(t *testing.T) {
	t.Parallel()
	t.Log("running TestK8sDashboardApi test")

	// Setup
	dbAPI := newObjectModelAPI(t)

	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	svcDomain := createServiceDomainWithLabels(t, dbAPI, tenantID, nil)
	svcDomainID := svcDomain.ID

	ctx1, _, _ := makeContext(tenantID, []string{})
	users := []model.User{}
	for i := 0; i < 5; i++ {
		users = append(users, createUserWithRole(t, dbAPI, tenantID, "USER"))
	}

	// Teardown
	defer func() {
		for _, u := range users {
			dbAPI.DeleteUser(ctx1, u.ID, nil)
		}
		dbAPI.DeleteServiceDomain(ctx1, svcDomainID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test GetViewonlyUsers ", func(t *testing.T) {
		t.Log("running GetViewonlyUsers test")
		userIDs := []string{users[0].ID, users[1].ID}
		err := dbAPI.AddViewonlyUsersToSD(ctx1, svcDomainID, userIDs)
		require.NoError(t, err)

		voUsers, err := dbAPI.GetViewonlyUsersForSD(ctx1, svcDomainID)
		require.NoError(t, err)
		require.Equal(t, 2, len(voUsers), "expect two viewonly users")
		if voUsers[0].ID == userIDs[0] {
			require.Equal(t, voUsers[1].ID, userIDs[1], "Expect viewonly user IDs to match")
		} else {
			require.Equal(t, voUsers[1].ID, userIDs[0], "Expect viewonly user IDs to match")
			require.Equal(t, voUsers[0].ID, userIDs[1], "Expect viewonly user IDs to match")
		}

		err = dbAPI.RemoveViewonlyUsersFromSD(ctx1, svcDomainID, []string{users[0].ID})
		require.NoError(t, err)
		voUsers, err = dbAPI.GetViewonlyUsersForSD(ctx1, svcDomainID)
		require.NoError(t, err)
		require.Equal(t, 1, len(voUsers), "Expect viewonly user number to match")
		require.Equal(t, voUsers[0].ID, userIDs[1], "Expect viewonly user IDs to match")

		userIDs = funk.Map(users[2:], func(user model.User) string { return user.ID }).([]string)
		err = dbAPI.AddViewonlyUsersToSD(ctx1, svcDomainID, userIDs)
		require.NoError(t, err)

		voUsers, err = dbAPI.GetViewonlyUsersForSD(ctx1, svcDomainID)
		require.NoError(t, err)
		require.Equal(t, 4, len(voUsers), "Expect viewonly user number to match")

		userIDs = funk.Map(users[1:], func(user model.User) string { return user.ID }).([]string)
		err = dbAPI.RemoveViewonlyUsersFromSD(ctx1, svcDomainID, userIDs)
		require.NoError(t, err)

		voUsers, err = dbAPI.GetViewonlyUsersForSD(ctx1, svcDomainID)
		require.NoError(t, err)
		require.Equal(t, 0, len(voUsers), "Expect viewonly user number to match")

	})
}
