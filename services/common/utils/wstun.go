package utils

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"crypto/md5"

	"encoding/json"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/golang/glog"
)

// Every websocket based ssh tunneling session to the edge will need
// to allocate a TCP port between WstunStartPort and WstunEndPort.
// This utility helps manage the allocation / release of such ports.
// In the case of scale out, this allocation must be global across
// all cloudmgmt pods, thus redis is used.

// use port between 20000 and 32767 for ssh tunneling
const (
	WstunStartPort = 20000
	WstunEndPort   = 32767
)

// WstunUtil - wstun util interface to support allocate / release port for wstun
type WstunUtil interface {
	// AllocatePort - allocate a port and store it in payload
	// Use redis if redisClient != nil (to support cloudmgmt scale out),
	// Otherwise use in-memory implementation.
	AllocatePort(redisClient *redis.Client, payload *model.WstunPayload, d time.Duration) error
	// ReleasePort, return port, error. If error != nil or port not found, return port=-1
	// Use redis if redisClient != nil (to support cloudmgmt scale out),
	// Otherwise use in-memory implementation.
	ReleasePort(redisClient *redis.Client, doc model.WstunTeardownRequestInternal, d time.Duration) (int32, error)
	// ClearExpiredPorts - clear and return a list of expired ports - for internal cleanup use
	ClearExpiredPorts(redisClient *redis.Client) []int32
}

// in-memory wstun struct
type wstunStruct struct {
	// port to tunneling record map
	port2TunnelingRecordMap map[uint32]tunnelingRecord
	// tunneling request to port map
	wstunRequest2PortMap map[model.WstunRequestInternal]uint32
	mx                   sync.Mutex
}

// tunneling record
type tunnelingRecord struct {
	TenantID        string
	ServiceDomainID string
	Expiration      int64
	Endpoint        string `json:"endpoint,omitempty"`
}

func getSalt(s string) string {
	if s == "" {
		return ""
	}
	data := []byte(s)
	return fmt.Sprintf("-%x", md5.Sum(data))[:10]
}

func salted(s, salt string) string {
	return fmt.Sprintf("%s%s", s, salt)
}

// NewWstunUtil - create instance of WstunUtil
func NewWstunUtil() WstunUtil {
	return &wstunStruct{
		port2TunnelingRecordMap: make(map[uint32]tunnelingRecord),
		wstunRequest2PortMap:    make(map[model.WstunRequestInternal]uint32),
	}
}

func (wstun *wstunStruct) ClearExpiredPorts(redisClient *redis.Client) []int32 {
	if redisClient != nil {
		return wstun.clearExpiredPortsRedis(redisClient)
	}
	return wstun.clearExpiredPortsInMem()
}

func (wstun *wstunStruct) AllocatePort(redisClient *redis.Client, payload *model.WstunPayload, d time.Duration) error {
	if redisClient != nil {
		return wstun.allocatePortRedis(redisClient, payload, d)
	}
	return wstun.allocatePortInMem(payload, d)
}

func (wstun *wstunStruct) ReleasePort(redisClient *redis.Client, doc model.WstunTeardownRequestInternal, d time.Duration) (int32, error) {
	if redisClient != nil {
		return wstun.releasePortRedis(redisClient, doc, d)
	}
	return wstun.releasePortInMem(doc)
}

// ==============================
// in-memory based implementation
// ==============================
func (wstun *wstunStruct) allocatePortInMem(payload *model.WstunPayload, d time.Duration) error {
	wstun.mx.Lock()
	defer wstun.mx.Unlock()
	expiration := getExpireNano(d)
	now := getEpocNano()
	// first check if there is an active session
	req := model.WstunRequestInternal{
		TenantID:        payload.TenantID,
		ServiceDomainID: payload.ServiceDomainID,
		Endpoint:        payload.Endpoint,
	}
	port, ok := wstun.wstunRequest2PortMap[req]
	if ok {
		rec, ok := wstun.port2TunnelingRecordMap[port]
		if ok {
			// see if session expired
			if rec.Expiration > now {
				// session still good
				payload.Expiration = expiration
				payload.Port = port
				// update expiration
				rec.Expiration = expiration
				wstun.port2TunnelingRecordMap[port] = rec
				return nil
			}
			// session expired, take it
			delete(wstun.wstunRequest2PortMap, req)
			delete(wstun.port2TunnelingRecordMap, port)
		} else {
			// should not happen
			glog.Warningf("<tenant_id=%s, edge_id=%s> ssh port %d allocated, but record not found", payload.TenantID, payload.ServiceDomainID, port)
			delete(wstun.wstunRequest2PortMap, req)
		}
	}
	// no active session, so proceed to allocate one
	err := setFreePort(wstun.port2TunnelingRecordMap, wstun.wstunRequest2PortMap, payload, now, expiration)
	if err != nil {
		return err
	}
	p := payload.Port
	wstun.wstunRequest2PortMap[req] = p
	glog.Infof("added session record %+v for port %d\n", wstun.port2TunnelingRecordMap[p], p)
	return nil
}

func (wstun *wstunStruct) releasePortInMem(doc model.WstunTeardownRequestInternal) (int32, error) {
	wstun.mx.Lock()
	defer wstun.mx.Unlock()
	rdoc := doc.ToRequest()
	port, ok := wstun.wstunRequest2PortMap[rdoc]
	if !ok {
		// not found
		return -1, nil
	}

	rec, ok := wstun.port2TunnelingRecordMap[port]
	if !ok {
		// should not happen
		glog.Warningf("<tenant_id=%s, edge_id=%s> ssh port %d allocated, but record not found", doc.TenantID, doc.ServiceDomainID, port)
		delete(wstun.wstunRequest2PortMap, rdoc)
		return -1, fmt.Errorf("<tenant_id=%s, edge_id=%s> ssh port %d allocated, but record not found", doc.TenantID, doc.ServiceDomainID, port)
	}
	if rec.TenantID != doc.TenantID || rec.ServiceDomainID != doc.ServiceDomainID {
		// should not happen
		glog.Warningf("<tenant_id=%s, edge_id=%s> ssh port %d allocated, but used by <tenant_id=%s, edge_id=%s>???", doc.TenantID, doc.ServiceDomainID, port, rec.TenantID, rec.ServiceDomainID)
		return -1, errcode.NewBadRequestError("mismatch")
	}
	delete(wstun.port2TunnelingRecordMap, port)
	delete(wstun.wstunRequest2PortMap, rdoc)
	return int32(port), nil
}

func (wstun *wstunStruct) clearExpiredPortsInMem() (ports []int32) {
	wstun.mx.Lock()
	defer wstun.mx.Unlock()
	// add extra minute for buffer
	expiration := getEpocNanoWithBuffer()
	for port, rec := range wstun.port2TunnelingRecordMap {
		if rec.Expiration < expiration {
			ports = append(ports, int32(port))
		}
	}
	if len(ports) != 0 {
		for _, port := range ports {
			delete(wstun.port2TunnelingRecordMap, uint32(port))
		}
		var rsToDel []model.WstunRequestInternal
		for r, pt := range wstun.wstunRequest2PortMap {
			ipt := int32(pt)
			for _, port := range ports {
				if ipt == port {
					rsToDel = append(rsToDel, r)
					break
				}
			}
		}
		if len(rsToDel) != 0 {
			for _, r := range rsToDel {
				delete(wstun.wstunRequest2PortMap, r)
			}
		}
	}
	return
}

// ==============================
// redis based implementation
// ==============================
const (
	globalLockMillis = 30000
	edgeLockMillis   = 30000
	globalLockKey    = "wstun.global.lock"
	globalDataKey    = "wstun.global.data"
)

// global key stores port -> (tenant, edge, expiration) map
// per edge key stores (tenant, edge) -> (port, map[public key hash]expiration) map

type tunnelingPort struct {
	Port      uint32
	KeyExpMap map[string]int64
}

func (tp tunnelingPort) isExpired() bool {
	expired := true
	now := getEpocNano()
	for _, e := range tp.KeyExpMap {
		if e > now {
			expired = false
			break
		}
	}
	return expired
}

type tunnelingRecordMap map[uint32]tunnelingRecord

func newTunnelingRecordMap() tunnelingRecordMap {
	return make(map[uint32]tunnelingRecord)
}

func (trm tunnelingRecordMap) MarshalBinary() ([]byte, error) {
	return json.Marshal(trm)
}

func (rec *tunnelingRecord) MarshalBinary() ([]byte, error) {
	return json.Marshal(*rec)
}

func (tp *tunnelingPort) MarshalBinary() ([]byte, error) {
	return json.Marshal(*tp)
}

func getWstunEdgeLockKey(tenantID, serviceDomainID, endpoint string) string {
	return fmt.Sprintf("wstun:lock:%s:%s", tenantID, salted(serviceDomainID, endpoint))
}
func getWstunEdgeDataKey(tenantID, serviceDomainID, endpoint string) string {
	return fmt.Sprintf("wstun:data:%s:%s", tenantID, salted(serviceDomainID, endpoint))
}

// TODO - enhance redis lock to include retry logic and timeout

func (wstun *wstunStruct) allocatePortRedis(redisClient *redis.Client, doc *model.WstunPayload, d time.Duration) error {
	// lock edge resource key
	edgeLockKey := getWstunEdgeLockKey(doc.TenantID, doc.ServiceDomainID, doc.Endpoint)
	edgeLockVal, ok := RedisLock(redisClient, edgeLockKey, edgeLockMillis)
	if !ok {
		return fmt.Errorf("failed to get wstun edge lock for tenant:%s, edge:%s", doc.TenantID, doc.ServiceDomainID)
	}
	defer RedisUnlock(redisClient, edgeLockKey, edgeLockVal)
	// find the port
	now := getEpocNano()
	expiration := getExpireNano(d)
	tp := &tunnelingPort{}
	cleanup := false
	allocateNew := true
	pubKeyHash := base.GetMD5Hash(doc.PublicKey)
	edgeDataKey := getWstunEdgeDataKey(doc.TenantID, doc.ServiceDomainID, doc.Endpoint)
	err := RedisUnmarshal(redisClient, edgeDataKey, tp)
	if err == nil {
		// found
		// see if session expired
		if tp.isExpired() {
			// session expired, set cleanup flag
			cleanup = true
		} else {
			// session still good, so reuse
			tp.KeyExpMap[*pubKeyHash] = expiration
			doc.Expiration = expiration
			doc.Port = tp.Port
			allocateNew = false
		}
	} else {
		if err != redis.Nil {
			return err
		}
	}
	// grab global lock
	globalLockVal, ok := RedisLock(redisClient, globalLockKey, globalLockMillis)
	if !ok {
		return fmt.Errorf("failed to get wstun global lock for tenant:%s, edge:%s", doc.TenantID, doc.ServiceDomainID)
	}
	defer RedisUnlock(redisClient, globalLockKey, globalLockVal)
	// load global port map
	trm := newTunnelingRecordMap()
	err = RedisUnmarshal(redisClient, globalDataKey, &trm)
	if err != nil {
		if err != redis.Nil {
			return err
		}
	}
	// if cleanup set, release old port
	if cleanup {
		err = redisClient.Del(edgeDataKey).Err()
		if err != nil {
			return err
		}
		// update trm to remove edge entry, then save it back
		x, ok := trm[tp.Port]
		if ok {
			if x.TenantID == doc.TenantID && x.ServiceDomainID == doc.ServiceDomainID {
				delete(trm, tp.Port)
			}
		}
	}

	if allocateNew {
		// allocate a port
		// claim the port in edge data and global data
		err = setFreePort(trm, nil, doc, now, expiration)
		if err != nil {
			return err
		}
		tp.Port = doc.Port
		tp.KeyExpMap = map[string]int64{*pubKeyHash: expiration}
	} else {
		// update
		trm[tp.Port] = tunnelingRecord{
			TenantID:        doc.TenantID,
			ServiceDomainID: doc.ServiceDomainID,
			Expiration:      doc.Expiration,
			Endpoint:        doc.Endpoint,
		}
	}
	err = redisClient.Set(globalDataKey, trm, 0).Err()
	if err != nil {
		return err
	}

	// don't give finite expiration for data key
	// we need the key to be around for clean up
	err = redisClient.Set(edgeDataKey, tp, 0).Err()
	if err != nil {
		redisClient.Del(edgeDataKey) // ignore error
		return err
	}
	return nil
}

func (wstun *wstunStruct) releasePortRedis(redisClient *redis.Client, doc model.WstunTeardownRequestInternal, d time.Duration) (int32, error) {
	glog.Infof("releasePortRedis: doc: %+v, duration: %s\n", doc, d)
	// lock edge resource key
	edgeLockKey := getWstunEdgeLockKey(doc.TenantID, doc.ServiceDomainID, doc.Endpoint)
	edgeLockVal, ok := RedisLock(redisClient, edgeLockKey, edgeLockMillis)
	if !ok {
		return -1, fmt.Errorf("releasePortRedis: failed to get wstun edge lock for tenant:%s, edge:%s", doc.TenantID, doc.ServiceDomainID)
	}
	defer RedisUnlock(redisClient, edgeLockKey, edgeLockVal)
	// find the port
	tp := &tunnelingPort{}
	pubKeyHash := base.GetMD5Hash(doc.PublicKey)
	edgeDataKey := getWstunEdgeDataKey(doc.TenantID, doc.ServiceDomainID, doc.Endpoint)
	err := RedisUnmarshal(redisClient, edgeDataKey, tp)
	if err != nil {
		if err == redis.Nil {
			// not found, ok
			glog.Infof("releasePortRedis: Got redis.Nil error in unmarshal\n")
			return -1, nil
		}
		return -1, err
	}
	glog.Infof("releasePortRedis: keymap before delete: %+v, pub key hash: %s\n", tp.KeyExpMap, *pubKeyHash)
	delete(tp.KeyExpMap, *pubKeyHash)
	glog.Infof("releasePortRedis: keymap after delete: %+v\n", tp.KeyExpMap)
	if !tp.isExpired() {
		glog.Infof("releasePortRedis: port %d is still in use\n", tp.Port)
		// other sessions still active, just save it
		// don't give finite expiration for data key
		// we need the key to be around for clean up
		err = redisClient.Set(edgeDataKey, tp, 0).Err()
		if err != nil {
			redisClient.Del(edgeDataKey) // ignore error
			return -1, err
		}
		// don't set port in this case,
		// since port is used to update k8s to remove port mapping
		return -1, nil
	}
	// port expired, so
	//   grab global lock
	//   clear the port and entry in global map
	globalLockVal, ok := RedisLock(redisClient, globalLockKey, globalLockMillis)
	if !ok {
		return -1, fmt.Errorf("releasePortRedis: failed to get wstun global lock for tenant:%s, edge:%s", doc.TenantID, doc.ServiceDomainID)
	}
	defer RedisUnlock(redisClient, globalLockKey, globalLockVal)
	// load global port map
	trm := newTunnelingRecordMap()
	err = RedisUnmarshal(redisClient, globalDataKey, &trm)
	if err != nil {
		return -1, err
	}
	// redis DEL edgeDataKey
	err = redisClient.Del(edgeDataKey).Err()
	if err != nil {
		return -1, err
	}
	// update trm to remove edge entry, then save it back
	x, ok := trm[tp.Port]
	if ok {
		if x.TenantID == doc.TenantID && x.ServiceDomainID == doc.ServiceDomainID {
			delete(trm, tp.Port)
			err = redisClient.Set(globalDataKey, trm, 0).Err()
			if err != nil {
				return -1, err
			}
			return int32(tp.Port), nil
		}
	}
	// port not owned by edge???
	return -1, fmt.Errorf("releasePortRedis: skip release port %d, port not owned by tenant:%s, edge:%s", tp.Port, doc.TenantID, doc.ServiceDomainID)
}

func (wstun *wstunStruct) clearExpiredPortsRedis(redisClient *redis.Client) (ports []int32) {
	// grab global lock
	globalLockVal, ok := RedisLock(redisClient, globalLockKey, globalLockMillis)
	if !ok {
		return
	}
	defer RedisUnlock(redisClient, globalLockKey, globalLockVal)
	// load global port map
	trm := newTunnelingRecordMap()
	err := RedisUnmarshal(redisClient, globalDataKey, &trm)
	if err != nil {
		return
	}
	// add extra minute for buffer
	expiration := getEpocNanoWithBuffer()
	for port, rec := range trm {
		if rec.Expiration < expiration {
			ports = append(ports, int32(port))
			edgeDataKey := getWstunEdgeDataKey(rec.TenantID, rec.ServiceDomainID, rec.Endpoint)
			redisClient.Del(edgeDataKey)
		}
	}
	if len(ports) != 0 {
		for _, port := range ports {
			pt := uint32(port)
			delete(trm, pt)
		}
	}
	return
}

// ==============================
// common utility functions
// ==============================

func getEpocNano() int64 {
	return time.Now().UTC().UnixNano()
}

func getExpireNano(d time.Duration) int64 {
	// now + 30min
	return getEpocNano() + int64(d/time.Nanosecond)
}

func getEpocNanoWithBuffer() int64 {
	// epoc nano + one extra minute for buffer
	return getEpocNano() + int64(time.Minute/time.Nanosecond)
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func getHashPort(s string) uint32 {
	return WstunStartPort + hash(s)%12763
}

func setFreePort(port2TunnelingRecordMap map[uint32]tunnelingRecord, wstunRequest2PortMap map[model.WstunRequestInternal]uint32, payload *model.WstunPayload, now, expiration int64) error {
	// no active session, so proceed to allocate one
	p0 := getHashPort(fmt.Sprintf("%s|%s%s", payload.TenantID, payload.ServiceDomainID, payload.Endpoint))
	var p uint32

	for p = p0; p < WstunEndPort; p++ {
		if isPortAvailable(port2TunnelingRecordMap, wstunRequest2PortMap, p, now) {
			break
		}
	}
	if p == WstunEndPort {
		for p = WstunStartPort; p < p0; p++ {
			if isPortAvailable(port2TunnelingRecordMap, wstunRequest2PortMap, p, now) {
				break
			}
		}
		if p == p0 {
			return fmt.Errorf("No free port")
		}
	}
	payload.Port = p
	payload.Expiration = expiration
	port2TunnelingRecordMap[p] = tunnelingRecord{
		TenantID:        payload.TenantID,
		ServiceDomainID: payload.ServiceDomainID,
		Expiration:      payload.Expiration,
		Endpoint:        payload.Endpoint,
	}
	return nil
}

// must be called after acquiring lock (for in-mem implementation)
func isPortAvailable(port2TunnelingRecordMap map[uint32]tunnelingRecord, wstunRequest2PortMap map[model.WstunRequestInternal]uint32, p uint32, now int64) bool {
	avail := false
	rec, ok := port2TunnelingRecordMap[p]
	if ok {
		if rec.Expiration < now {
			// record expired, take it
			if wstunRequest2PortMap != nil {
				delete(wstunRequest2PortMap, model.WstunRequestInternal{
					TenantID:        rec.TenantID,
					ServiceDomainID: rec.ServiceDomainID,
					Endpoint:        rec.Endpoint,
				})
			}
			delete(port2TunnelingRecordMap, p)
			avail = true
		} else {
			glog.Infof("port %d already in use by record: %+v\n", p, rec)
		}
	} else {
		// not in use, take it
		avail = true
	}
	return avail
}
