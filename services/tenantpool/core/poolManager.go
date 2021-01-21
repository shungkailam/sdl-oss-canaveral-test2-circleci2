package core

// PoolManager uses BookKeeper to manage pool level, registration etc
import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/metrics"
	cloudmgmtmodel "cloudservices/common/model"
	"cloudservices/tenantpool/config"
	"cloudservices/tenantpool/model"
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// InfraType is the edge type
	InfraType = string(cloudmgmtmodel.CloudTargetType)
	// InfraName is the default edge name
	InfraName = "xi-cloud"
	// scanTenantClaimsTimeout is the timeout for each DB scan call
	scanTenantClaimsTimeout = time.Minute * 5
	// Scanner lock name
	scannerLockName = "tenantpool-scanner-lock"
)

// TenantPoolManager manages the pool of tenants with corresponding edges
type TenantPoolManager struct {
	cancel context.CancelFunc
	// BookKeeper keeps track of availability of tenants and edges
	bookKeeper *BookKeeper
	// Instance of the EdgeProvisioner
	edgeProvisioner model.EdgeProvisioner
	// Instance of the AuditLogManager
	auditLogManager *AuditLogManager
	// Scanner notification
	scannerDone *sync.WaitGroup
	// Cache of pool calculation results for registrations
	poolStats *sync.Map
}

type PoolStats struct {
	mutex           *sync.Mutex
	poolSize        int
	st              float64
	downsizeCounter int
	samplingCounter int
	lastRequestTime time.Time
}

func NewPoolStats(initialAvg float64, availableCount, samplingCounter int) *PoolStats {
	return &PoolStats{
		samplingCounter: samplingCounter,
		poolSize:        availableCount,
		st:              initialAvg,
		mutex:           &sync.Mutex{},
		lastRequestTime: base.RoundedNow(),
	}
}

func (poolStats *PoolStats) GetWeightedAvg() float64 {
	poolStats.mutex.Lock()
	defer poolStats.mutex.Unlock()
	return poolStats.st
}

func (poolStats *PoolStats) GetPoolSize() int {
	poolStats.mutex.Lock()
	defer poolStats.mutex.Unlock()
	return poolStats.poolSize
}

func (manager *TenantPoolManager) GetPoolStats(ctx context.Context, registrationID string) *PoolStats {
	value, ok := manager.poolStats.Load(registrationID)
	if !ok {
		return nil
	}
	return value.(*PoolStats)
}

// isTransitionState checks if the state is a transition state
func isTransitionState(state string) bool {
	return strings.HasSuffix(state, "ING")
}

// NewTenantPoolManagerWithRedisClient instantiates TenantPoolManager accepting the redisClient for internal caching
func NewTenantPoolManagerWithRedisClient(edgeProvisioner model.EdgeProvisioner, redisClient *redis.Client) (*TenantPoolManager, error) {
	if edgeProvisioner == nil {
		return nil, errors.New("BookKeeper is not set")
	}
	dbURL, err := base.GetDBURL(*config.Cfg.SQLDialect, *config.Cfg.SQLDB, *config.Cfg.SQLUser, *config.Cfg.SQLPassword, *config.Cfg.SQLHost, *config.Cfg.SQLPort, false)
	if err != nil {
		glog.Errorf("Failed to create book keeper instance. Error: %s", err.Error())
		return nil, err
	}
	roDbURL := dbURL
	if config.Cfg.SQLReadOnlyHost != nil && len(*config.Cfg.SQLReadOnlyHost) > 0 {
		roDbURL, err = base.GetDBURL(*config.Cfg.SQLDialect, *config.Cfg.SQLDB, *config.Cfg.SQLUser, *config.Cfg.SQLPassword, *config.Cfg.SQLReadOnlyHost, *config.Cfg.SQLPort, false)
		if err != nil {
			return nil, err
		}
	}
	dbAPI, err := base.NewDBObjectModelAPI(*config.Cfg.SQLDialect, dbURL, roDbURL, redisClient)
	if err != nil {
		glog.Errorf("Failed to create db object model API instance. Error: %s", err.Error())
		return nil, err
	}
	auditLogManager := &AuditLogManager{DBObjectModelAPI: dbAPI}
	bookKeeper := &BookKeeper{DBObjectModelAPI: dbAPI}
	tenantPoolManager := &TenantPoolManager{auditLogManager: auditLogManager, bookKeeper: bookKeeper, edgeProvisioner: edgeProvisioner, scannerDone: &sync.WaitGroup{}, poolStats: &sync.Map{}}
	if *config.Cfg.EnableScanner {
		ctx := context.Background()
		ctx, tenantPoolManager.cancel = context.WithCancel(ctx)
		tenantPoolManager.scheduleScan(ctx)
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			tenantPoolManager.Close()
		}()
	}
	return tenantPoolManager, nil
}

// NewTenantPoolManager instantiates TenantPoolManager
func NewTenantPoolManager(edgeProvisioner model.EdgeProvisioner) (*TenantPoolManager, error) {
	return NewTenantPoolManagerWithRedisClient(edgeProvisioner, nil)
}

// Close cleans up
func (manager *TenantPoolManager) Close() error {
	if manager.cancel != nil {
		manager.cancel()
		manager.scannerDone.Wait()
	}
	return manager.bookKeeper.Close()
}

// GetBookKeeper returns the internal BookKeeper instance.
func (manager *TenantPoolManager) GetBookKeeper() *BookKeeper {
	return manager.bookKeeper
}

// GetAuditLogManager returns the internal AuditLogManager instance.
func (manager *TenantPoolManager) GetAuditLogManager() *AuditLogManager {
	return manager.auditLogManager
}

// GetEdgeProvisioner returns the internal EdgeProvisioner instance.
func (manager *TenantPoolManager) GetEdgeProvisioner() model.EdgeProvisioner {
	return manager.edgeProvisioner
}

// CreateTenantClaim creates a tenant in Xi IoT and also provisions edges for each tenant
// If the tenantID is empty, a tenantID is created internally
func (manager *TenantPoolManager) CreateTenantClaim(ctx context.Context, registrationID, tenantID string) (*model.TenantClaim, error) {
	return manager.bookKeeper.CreateTenantClaim(ctx, registrationID, tenantID, func(registration *model.Registration, tenantClaim *model.TenantClaim) error {
		regConfig, err := registration.GetConfig(ctx)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to create edge for tenant %s. Error: %s"), tenantClaim.ID, err.Error())
			return err
		}
		if regConfig.GetVersionInfo().Version == model.RegConfigV1 {
			configV1 := regConfig.(*model.RegistrationConfigV1)
			// Always deploy for trials - backward compatibility
			deployApps := configV1.DeployApps || tenantClaim.Trial
			for i := 0; i < configV1.EdgeCount; i++ {
				createEdgeConfig := &model.CreateEdgeConfig{
					TenantID:           tenantClaim.ID,
					SystemUser:         tenantClaim.SystemUser,
					SystemPassword:     tenantClaim.SystemPassword,
					DeployApp:          deployApps,
					DatapipelineDeploy: deployApps,
					DatasourceDeploy:   deployApps,
					AppChartVersion:    *config.Cfg.AppChartVersion,
					InstanceType:       configV1.InstanceType,
					Tags: []string{
						fmt.Sprintf("user=%s", *config.Cfg.Namespace),
						fmt.Sprintf("registration=%s", registrationID),
						fmt.Sprintf("tenant=%s", tenantClaim.ID),
						fmt.Sprintf("chart-version=%s", *config.Cfg.AppChartVersion),
						fmt.Sprintf("trial=%t", tenantClaim.Trial),
					},
				}
				createEdgeConfig.Name = InfraName
				if i > 0 {
					createEdgeConfig.Name = fmt.Sprintf("%s-%d", InfraName, i)
				}
				edgeInfo, err := manager.edgeProvisioner.CreateEdge(ctx, createEdgeConfig)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Failed to create edge for tenant %s. Error: %s"), tenantClaim.ID, err.Error())
					return err
				}
				edgeContext := &model.EdgeContext{ID: edgeInfo.ContextID, State: edgeInfo.State, Type: InfraType}
				tenantClaim.EdgeContexts = append(tenantClaim.EdgeContexts, edgeContext)
			}
		}
		return nil
	})
}

// ensureTenantCapacity makes sure that the min pool size of tenants is maintained
func (manager *TenantPoolManager) ensureTenantCapacity(ctx context.Context, registration *model.Registration, availableTenants, assignedTenants, pendingCreateTenants []*model.TenantClaim) error {
	regConfig, err := registration.GetConfig(ctx)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get registration config from registration %+v to ensure tenant capacity. Error: %s"), registration, err.Error())
	}

	assignedTenantCount := len(assignedTenants)
	availableTenantCount := len(availableTenants)
	pendingCreateTenantCount := len(pendingCreateTenants)

	minTenantPoolSize := 0
	maxTenantPoolSize := 0
	maxPendingTenantCount := 0
	if regConfig.GetVersionInfo().Version == model.RegConfigV1 {
		configV1 := regConfig.(*model.RegistrationConfigV1)
		minTenantPoolSize = configV1.MinTenantPoolSize
		maxTenantPoolSize = configV1.MaxTenantPoolSize
		maxPendingTenantCount = configV1.MaxPendingTenantCount
	}
	deletableTenants := 0
	addableTenants := 0
	totalUnclaimedTenants := availableTenantCount + pendingCreateTenantCount
	totalPotentialTenants := totalUnclaimedTenants + assignedTenantCount

	if *config.Cfg.EnablePoolStats {
		// Calculate the pool stats
		poolStats, err := manager.CalculatePoolStats(ctx, registration, availableTenantCount)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to calculate pool stats for registration %s. Error: %s"), registration.ID, err.Error())
			return err
		}
		minTenantPoolSize = poolStats.GetPoolSize()
		glog.Infof(base.PrefixRequestID(ctx, "Dynamic pool sizing is enabled - current level %d"), minTenantPoolSize)
	}
	if totalPotentialTenants > maxTenantPoolSize {
		// Consider deletion because the total potential is more than the max pool size
		// Hard limit violation
		deletableTenants = totalPotentialTenants - maxTenantPoolSize
	} else if availableTenantCount > minTenantPoolSize {
		// Soft limit violation..pending ones are not considered
		deletableTenants = availableTenantCount - minTenantPoolSize
	} else if totalUnclaimedTenants < minTenantPoolSize {
		// Consider addition because the total unclaimed (available + pending) is less than min pool size
		// AND the total potential (available + pending + assigned) is less than the max pool size
		maxAddableTenants := maxTenantPoolSize - totalPotentialTenants
		addableTenants = minTenantPoolSize - totalUnclaimedTenants
		if addableTenants > maxAddableTenants {
			// Optimization - this avoids chatter - creating too many first and then deletion in the next cycle
			// e.g on recreate, we can have min=10, max=10, unclaimed=0, assigned=9 => min - unclaimed = 10 can be created in the first scan.
			// When the 10 tenants are created, the deletion kicks in because assigned + unclaimed ones = 19 > 10.
			// Max addable in this case is 10 - 9 = 1 which is the right number.
			// This happens when the difference between min and max is small and the unclaimed suddenly drops due to sudden signup or tenant recreation
			addableTenants = maxAddableTenants
		}
	} else {
		glog.Infof(base.PrefixRequestID(ctx, "Pool size %d is within the limits [%d, %d] for registration %s"), totalPotentialTenants, minTenantPoolSize, maxTenantPoolSize, registration.ID)
		return nil
	}
	if deletableTenants > 0 {
		if deletableTenants > availableTenantCount {
			// You can delete only what is available
			deletableTenants = availableTenantCount
		}
		deletedTenants := 0
		glog.Infof(base.PrefixRequestID(ctx, "Deleting %d tenants to meet max pool size for registration %+v"), deletableTenants, registration)
		for i := 0; i < availableTenantCount; i++ {
			if deletedTenants >= deletableTenants {
				break
			}
			tenantClaim := availableTenants[i]
			tenantClaim.State = Deleting
			// TODO delete pending first?
			err = manager.bookKeeper.UpdateTenantClaimTxn(ctx, registration, tenantClaim)
			if err == nil {
				deletedTenants++
			} else {
				glog.Infof(base.PrefixRequestID(ctx, "Failed to delete tenant %s while downsizing for registration %+v. Error: %s"), tenantClaim.ID, registration, err.Error())
			}
		}
		glog.Infof(base.PrefixRequestID(ctx, "Set to decrease tenant pool by %d out of %d for registration %+v"), deletedTenants, deletableTenants, registration)
	} else if addableTenants > 0 {
		// Cannot add more
		addablePendingTenants := maxPendingTenantCount - pendingCreateTenantCount
		if addableTenants > addablePendingTenants {
			addableTenants = addablePendingTenants
		}
		glog.Infof(base.PrefixRequestID(ctx, "Acquiring %d new tenants"), addableTenants)
		errMsg := []string{}
		for i := 0; i < addableTenants; i++ {
			_, err := manager.CreateTenantClaim(ctx, registration.ID, "")
			if err != nil {
				errMsg = append(errMsg, err.Error())
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to create tenant to ensure capacity for registration %+v. Error: %s"), registration, err.Error())
			}
		}
		if len(errMsg) > 0 {
			return fmt.Errorf(base.PrefixRequestID(ctx, "Error occurred in ensuring tenantClaim capacity. Error: %s"), strings.Join(errMsg, ", "))
		}
	}
	return nil
}

// scheduleScan starts the scheduler
func (manager *TenantPoolManager) scheduleScan(ctx context.Context) {
	base.ScheduleJob(ctx, "TenantPoolScanner", func(canCtx context.Context) {
		manager.scannerDone.Add(1)
		defer func() {
			manager.scannerDone.Done()
			select {
			case <-ctx.Done():
				return
			default:
				manager.scheduleScan(ctx)
			}
		}()
		// Check for cancelled context because the check for waitgroup
		// to be zero could have done before this method is invoked
		select {
		case <-ctx.Done():
			return
		default: // continue
		}
		// Just get the ctx values and create a per scan timeout
		calleeCtx, cancelFn := context.WithTimeout(context.WithValue(canCtx, base.RequestIDKey, base.GetUUID()), scanTenantClaimsTimeout)
		defer cancelFn()
		glog.Infof(base.PrefixRequestID(calleeCtx, "Running tenantClaims scanner at %s"), time.Now().UTC().Format("Mon Jan _2 15:04:05 2006"))
		registrations, _, err := manager.bookKeeper.GetRegistrations(ctx, "", []string{}, nil)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to schedule tenantClaims scanner. Error: %s"), err.Error())
		}
		for idx := 0; idx < len(registrations); idx++ {
			availableTenants := []*model.TenantClaim{}
			assignedTenants := []*model.TenantClaim{}
			pendingCreateTenants := []*model.TenantClaim{}
			registration := registrations[idx]
			tenantCount := 0
			failedTenantsCount := 0
			_, err = manager.bookKeeper.ScanTenantClaims(calleeCtx, registration.ID, "", []string{}, nil, func(registration *model.Registration, tenantClaim *model.TenantClaim) error {
				tenantCount++
				if registration.State == Deleting {
					glog.Infof(base.PrefixRequestID(ctx, "Registration is in %s state. Triggering tenant deletion for %+v"), registration.State, tenantClaim)
					if tenantClaim.State != Deleting {
						tenantClaim.State = Deleting
						err = manager.bookKeeper.UpdateTenantClaimTxn(calleeCtx, registration, tenantClaim)
						if err != nil {
							// Ignore error
							glog.Errorf(base.PrefixRequestID(calleeCtx, "Failed to update tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
						}
					}
				}
				switch tenantClaim.State {
				case Creating:
					manager.handleCreatingState(calleeCtx, registration, tenantClaim)
					// Consider only trial for capacity calculation
					if tenantClaim.Trial {
						pendingCreateTenants = append(pendingCreateTenants, tenantClaim)
					}
				case Deleting:
					manager.handleDeletingState(calleeCtx, registration, tenantClaim)
				case Failed:
					manager.handleFailedState(calleeCtx, registration, tenantClaim)
					failedTenantsCount++
				case Available:
					manager.handleAvailableState(calleeCtx, registration, tenantClaim)
					// Consider only trial for capacity calculation
					if tenantClaim.Trial {
						availableTenants = append(availableTenants, tenantClaim)
					}
				case Assigned:
					manager.handleAssignedState(calleeCtx, registration, tenantClaim)
					// Consider only trial for capacity calculation
					if tenantClaim.Trial {
						assignedTenants = append(assignedTenants, tenantClaim)
					}
				case Reserved:
					manager.handleReservedState(calleeCtx, registration, tenantClaim)
				}
				return nil
			})
			if err != nil {
				glog.Errorf(base.PrefixRequestID(calleeCtx, "Failed to scan tenantClaims for registration %+v. Error: %s"), registration, err.Error())
				continue
			}
			creatingTenantsCount := len(pendingCreateTenants)
			assignedTenantsCount := len(assignedTenants)
			availableTenantsCount := len(availableTenants)

			// Prometheus stats
			metrics.CreatingTrialTenants.With(prometheus.Labels{
				"registration": registration.ID,
			}).Set(float64(creatingTenantsCount))

			metrics.AssignedTrialTenants.With(prometheus.Labels{
				"registration": registration.ID,
			}).Set(float64(assignedTenantsCount))

			metrics.AvailableTrialTenants.With(prometheus.Labels{
				"registration": registration.ID,
			}).Set(float64(availableTenantsCount))

			metrics.FailedTrialTenants.With(prometheus.Labels{
				"registration": registration.ID,
			}).Set(float64(failedTenantsCount))

			if registration.State == Deleting {
				if tenantCount == 0 {
					glog.Infof(base.PrefixRequestID(calleeCtx, "Deleting registration %+v"), registration)
					err = manager.bookKeeper.DeleteRegistration(calleeCtx, registration.ID)
					if err != nil {
						glog.Warningf(base.PrefixRequestID(ctx, "Failed to delete registration %+v. Error: %s"), registration, err.Error())
					}
				}
			} else {
				glog.Infof(base.PrefixRequestID(calleeCtx, "Pool level - available: %d, pending: %d, assigned: %d"), availableTenantsCount, creatingTenantsCount, assignedTenantsCount)
				err = manager.ensureTenantCapacity(calleeCtx, registration, availableTenants, assignedTenants, pendingCreateTenants)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(calleeCtx, "Failed in ensuring tenant capacity for registration %+v. Error: %s"), registration, err.Error())
				}
			}
		}
	}, *config.Cfg.TenantPoolScanDelay)
}

// addResourcesToTenantClaim adds resources in edgeInfo to tenantClaim
func addResourcesToTenantClaim(ctx context.Context, tenantClaim *model.TenantClaim, edgeContext *model.EdgeContext, edgeInfo *model.EdgeInfo) {
	if edgeInfo.Edge == nil || len(edgeInfo.Edge.ID) == 0 {
		glog.Warningf(base.PrefixRequestID(ctx, "No edge ID received for edge context %+v"), edgeContext)
		return
	}
	edgeContext.EdgeID = base.StringPtr(edgeInfo.Edge.ID)
	if edgeInfo.Resources == nil || len(edgeInfo.Resources) == 0 {
		glog.Warningf(base.PrefixRequestID(ctx, "No resources received for edge context %+v"), edgeContext)
		return
	}
	for resourceID, resource := range edgeInfo.Resources {
		if len(resourceID) == 0 || resource == nil {
			glog.Warningf(base.PrefixRequestID(ctx, "Invalid resource found for edge info %+v"), edgeInfo)
			continue
		}
		var ok bool
		var tenantResource *model.TenantResource
		if tenantClaim.Resources == nil {
			tenantClaim.Resources = map[string]*model.TenantResource{}
		}
		if tenantResource, ok = tenantClaim.Resources[resourceID]; !ok {
			tenantResource = &model.TenantResource{Resource: *resource}
			tenantClaim.Resources[resourceID] = tenantResource
		}
		tenantResource.EdgeIDs = append(tenantResource.EdgeIDs, edgeInfo.Edge.ID)
	}
}

func (manager *TenantPoolManager) CalculatePoolStats(ctx context.Context, registration *model.Registration, availableCount int) (*PoolStats, error) {
	regConfig, err := registration.GetConfig(ctx)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get registration config %s. Error: %s"), registration.ID, err.Error())
		return nil, err
	}
	minTenantPoolSize := 0
	maxTenantPoolSize := 0

	if regConfig.GetVersionInfo().Version == model.RegConfigV1 {
		configV1 := regConfig.(*model.RegistrationConfigV1)
		minTenantPoolSize = configV1.MinTenantPoolSize
		maxTenantPoolSize = configV1.MaxTenantPoolSize
	}
	initialAvg := float64(availableCount) / float64(*config.Cfg.PoolStatsTimeFactor)
	value, found := manager.poolStats.LoadOrStore(registration.ID, NewPoolStats(initialAvg, availableCount, 0))
	poolStats := value.(*PoolStats)
	if !found {
		return poolStats, nil
	}
	poolStats.mutex.Lock()
	defer poolStats.mutex.Unlock()
	poolStats.samplingCounter++
	if poolStats.samplingCounter < *config.Cfg.PoolStatsSamplingInterval {
		return poolStats, nil
	}
	requestCount, lastCreatedAt, err := manager.auditLogManager.GetAuditLogCount(ctx, &model.AuditLog{
		RegistrationID: registration.ID,
		Action:         model.AuditLogReserveTenantAction,
		CreatedAt:      poolStats.lastRequestTime,
	})
	if err != nil {
		return nil, err
	}
	glog.Infof(base.PrefixRequestID(ctx, "Current number of requests: %d, time: %+v"), requestCount, lastCreatedAt)
	err = manager.CalculatePoolStatsHelper(ctx, poolStats, requestCount, minTenantPoolSize, maxTenantPoolSize)
	if err != nil {
		return nil, err
	}
	glog.Infof(base.PrefixRequestID(ctx, "Calculated pool stats - %+v"), poolStats)
	poolStats.samplingCounter = 0
	poolStats.lastRequestTime = lastCreatedAt
	return poolStats, nil
}

func (manager *TenantPoolManager) CalculatePoolStatsHelper(ctx context.Context, poolStats *PoolStats, requestCount, minTenantPoolSize, maxTenantPoolSize int) error {
	weight := *config.Cfg.PoolStatsWeight
	factor := *config.Cfg.PoolStatsTimeFactor
	downsizeDelay := *config.Cfg.PoolStatsDownsizeDelay
	// Weightted average in poolStats.samplingCounter minutes (scan interval is 1)
	st := weight*float64(requestCount) + (1.0-weight)*poolStats.st
	poolSize := int(math.Floor((factor * st) / float64(poolStats.samplingCounter)))
	if poolSize < minTenantPoolSize {
		poolSize = minTenantPoolSize
	}
	if poolSize > maxTenantPoolSize {
		poolSize = maxTenantPoolSize
	}
	if poolSize < poolStats.poolSize {
		if poolStats.downsizeCounter > downsizeDelay {
			// Do not decrease suddenly
			poolStats.poolSize = poolStats.poolSize - int(math.Min(float64(*config.Cfg.PoolStatsDownsizeLimit), float64(poolStats.poolSize-poolSize)))
			poolStats.downsizeCounter = 0
		} else {
			poolStats.downsizeCounter++
		}
	} else {
		poolStats.poolSize = poolSize
		poolStats.downsizeCounter = 0
	}
	poolStats.st = st
	return nil
}

// handleCreatingState handles tenant state of PENDING_CREATE
func (manager *TenantPoolManager) handleCreatingState(ctx context.Context, registration *model.Registration, tenantClaim *model.TenantClaim) error {
	errMsg := []string{}
	createdEdgeCount := 0
	recordUpdated := false
	for i := 0; i < len(tenantClaim.EdgeContexts); i++ {
		edgeContext := tenantClaim.EdgeContexts[i]
		if isTransitionState(edgeContext.State) {
			if time.Since(edgeContext.UpdatedAt) > *config.Cfg.EdgeProvisionTimeout {
				// Mark it FAILED to initiate the deletion
				tenantClaim.State = Failed
				edgeContext.State = Failed
				recordUpdated = true
				glog.Errorf(base.PrefixRequestID(ctx, "Timeout creating the edges for tenant %s"), tenantClaim.ID)
				break
			}
			// Need to determine the next state for the transition state
			edgeInfo, err := manager.edgeProvisioner.GetEdgeStatus(ctx, tenantClaim.ID, edgeContext.ID)
			if err != nil {
				errMsg = append(errMsg, err.Error())
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to get status for edge %s for registration %s and tenant %s. Error: %s"), edgeContext.ID, registration.ID, tenantClaim.ID, err.Error())
				// Update in the next iteration
				continue
			}
			if edgeContext.State != edgeInfo.State {
				glog.Infof(base.PrefixRequestID(ctx, "State change detected from %s to %s for edge %s in registration %s and tenant %s"), edgeContext.State, edgeInfo.State, edgeContext.ID, registration.ID, tenantClaim.ID)
				recordUpdated = true
			}
			edgeContext.State = edgeInfo.State
			if edgeInfo.State == Created {
				// Add the reported resources
				addResourcesToTenantClaim(ctx, tenantClaim, edgeContext, edgeInfo)
			}
		} else {
			glog.V(3).Infof(base.PrefixRequestID(ctx, "Edge context %+v for tenant ID %s is not in transition state"), edgeContext, tenantClaim.ID)
		}
		switch edgeContext.State {
		case Created:
			createdEdgeCount++
		case Failed:
			tenantClaim.State = Failed
		// Not supposed to happen but good to add
		case Deleted:
			tenantClaim.State = Failed
		}
	}
	// All the created
	if createdEdgeCount == len(tenantClaim.EdgeContexts) {
		if tenantClaim.Trial {
			tenantClaim.State = Available
		} else {
			tenantClaim.State = Assigned
		}
		// Edge onboarding done for the tenant.
		// Delete the system user
		_, err := DeleteUserByEmail(ctx, tenantClaim.SystemUser)
		if err != nil {
			if _, ok := err.(*errcode.RecordNotFoundError); ok {
				// Already deleted
				err = nil
			} else {
				// If error is returned, it will be retried
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to handle pending create state for tenantClaim %+v. System user deletion failed. Errors: %s"), tenantClaim, err.Error())
				return err
			}
		}
		recordUpdated = true
	}
	if recordUpdated {
		err := manager.bookKeeper.UpdateTenantClaimTxn(ctx, registration, tenantClaim)
		if err != nil {
			errMsg = append(errMsg, err.Error())
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
		}
	}
	var err error
	if len(errMsg) > 0 {
		err = fmt.Errorf(base.PrefixRequestID(ctx, "Failed to handle pending create state. Errors: %s"), strings.Join(errMsg, "[n] "))
		glog.Error(err)
	}
	return err
}

func (manager *TenantPoolManager) handleDeletingState(ctx context.Context, registration *model.Registration, tenantClaim *model.TenantClaim) error {
	errMsg := []string{}
	recordUpdated := false
	activeEdgeContexts := []*model.EdgeContext{}
	var edgeInfo *model.EdgeInfo
	var err error
	for i := 0; i < len(tenantClaim.EdgeContexts); i++ {
		edgeContext := tenantClaim.EdgeContexts[i]
		// If everything is good, deletion timeout must not happen.
		// We have seen it happening when bott is just deployed as it does not keep the state.
		isDeletionTimedOut := base.RoundedNow().Sub(tenantClaim.UpdatedAt) > *config.Cfg.EdgeDeletionTimeout
		if isTransitionState(edgeContext.State) && !isDeletionTimedOut {
			// Check status as long as it is in transition state and the deletion has not timed out
			// The status can be in CREATING OR DELETING state. Delete is not called in transition state.
			edgeInfo, err = manager.edgeProvisioner.GetEdgeStatus(ctx, tenantClaim.ID, edgeContext.ID)
			if err != nil {
				activeEdgeContexts = append(activeEdgeContexts, edgeContext)
				errMsg = append(errMsg, err.Error())
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to get status for edge %s for tenant %s. Error: %s"), edgeContext.ID, tenantClaim.ID, err.Error())
				continue
			}
		} else {
			glog.V(3).Infof(base.PrefixRequestID(ctx, "Edge context %+v for tenant ID %s is not in transition state"), edgeContext, tenantClaim.ID)
			// Can be in FAILED or CREATED state for edge
			// Invoke edge delete again on time out or the state is not in transition
			edgeInfo, err = manager.edgeProvisioner.DeleteEdge(ctx, tenantClaim.ID, edgeContext.ID)
			if err != nil {
				activeEdgeContexts = append(activeEdgeContexts, edgeContext)
				errMsg = append(errMsg, err.Error())
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete edge %s for tenant %s. Error: %s"), edgeContext.ID, tenantClaim.ID, err.Error())
				continue
			}
			if isDeletionTimedOut {
				// Update the record such that the updated_at time is updated in the DB
				// Next read should pick up the updated time
				recordUpdated = true
			}
		}
		if edgeContext.State != edgeInfo.State {
			glog.Infof(base.PrefixRequestID(ctx, "State change detected from %s to %s for edge %s in registration %s and tenant %s"), edgeContext.State, edgeInfo.State, edgeContext.ID, registration.ID, tenantClaim.ID)
			recordUpdated = true
		}
		edgeContext.State = edgeInfo.State
		if edgeInfo.State != Deleted {
			activeEdgeContexts = append(activeEdgeContexts, edgeContext)
		}
	}
	if len(activeEdgeContexts) == 0 {
		glog.Infof(base.PrefixRequestID(ctx, "Performing post delete edges for tenant %s"), tenantClaim.ID)
		err = manager.edgeProvisioner.PostDeleteEdges(ctx, tenantClaim.ID)
		if err != nil {
			errMsg = append(errMsg, err.Error())
		} else {
			err = manager.bookKeeper.RenameSerialNumbers(ctx, tenantClaim)
			if err != nil {
				errMsg = append(errMsg, err.Error())
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to rename node serial numbers %+v. Error: %s"), tenantClaim, err.Error())
			} else {
				err = manager.bookKeeper.DeleteTenantClaim(ctx, tenantClaim)
				if err != nil {
					errMsg = append(errMsg, err.Error())
					glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
				}
			}
		}
	} else if recordUpdated {
		tenantClaim.EdgeContexts = activeEdgeContexts
		err = manager.bookKeeper.UpdateTenantClaimTxn(ctx, registration, tenantClaim)
		if err != nil {
			errMsg = append(errMsg, err.Error())
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenant pool %+v. Error: %s"), tenantClaim, err.Error())
		}
	}
	err = nil
	if len(errMsg) > 0 {
		err = fmt.Errorf(base.PrefixRequestID(ctx, "Failed to handle pending delete state. Errors: %s"), strings.Join(errMsg, "[n] "))
		glog.Error(err)
	}
	return err
}

// If the tenant state is failed, the edges can be in pending delete or failed or created state
func (manager *TenantPoolManager) handleFailedState(ctx context.Context, registration *model.Registration, tenantClaim *model.TenantClaim) error {
	tenantClaim.State = Deleting
	err := manager.bookKeeper.UpdateTenantClaimTxn(ctx, registration, tenantClaim)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
	}
	return err
}

func (manager *TenantPoolManager) handleReservedState(ctx context.Context, registration *model.Registration, tenantClaim *model.TenantClaim) error {
	if time.Since(tenantClaim.UpdatedAt) > *config.Cfg.ReserveStateExpiry {
		var err error
		if tenantClaim.Trial {
			// The tenant has not been claimed yet
			tenantClaim.State = Available
		} else {
			// The tenant is already claimed
			tenantClaim.State = Assigned
			defer func() {
				auditLog := &model.AuditLog{TenantID: tenantClaim.ID, RegistrationID: tenantClaim.RegistrationID, Actor: model.AuditLogSystemActor, Action: model.AuditLogConfirmTenantAction}
				manager.auditLogManager.CreateAuditLogHelper(ctx, err, auditLog, true)
			}()
		}
		err = manager.bookKeeper.UpdateTenantClaimTxn(ctx, registration, tenantClaim)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenantClaim %+v after reservation time-out. Error: %s"), tenantClaim, err.Error())
		}
		return err
	}
	return nil
}

func (manager *TenantPoolManager) handleAssignedState(ctx context.Context, registration *model.Registration, tenantClaim *model.TenantClaim) error {
	if tenantClaim == nil || !tenantClaim.Trial || tenantClaim.AssignedAt == nil || tenantClaim.ExpiresAt == nil {
		return nil
	}
	var err error
	if time.Since(*tenantClaim.ExpiresAt) <= 0 {
		return nil
	}
	defer func() {
		auditLog := &model.AuditLog{TenantID: tenantClaim.ID, RegistrationID: tenantClaim.RegistrationID, Actor: model.AuditLogSystemActor, Action: model.AuditLogDeleteTenantAction}
		manager.auditLogManager.CreateAuditLogHelper(ctx, err, auditLog, true)
	}()
	glog.V(3).Infof(base.PrefixRequestID(ctx, "Expiring tenantClaim %+v"), tenantClaim)
	tenantClaim.State = Deleting
	err = manager.bookKeeper.UpdateTenantClaimTxn(ctx, registration, tenantClaim)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenantClaim %+v after trial expiry. Error: %s"), tenantClaim, err.Error())
		return err
	}
	glog.V(3).Infof(base.PrefixRequestID(ctx, "Set to expire tenantClaim %+v"), tenantClaim)
	return nil
}

// Not implemented
func (manager *TenantPoolManager) handleAvailableState(ctx context.Context, registration *model.Registration, tenantClaim *model.TenantClaim) error {
	return nil
}
