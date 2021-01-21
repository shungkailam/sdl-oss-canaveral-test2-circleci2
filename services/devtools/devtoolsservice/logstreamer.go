package devtoolsservice

import (
	"cloudservices/devtools/generated/swagger/models"

	"github.com/golang/glog"
)

type LogStreamer struct {
	redisManager *RedisManager
}

func NewLogStreamer(redisManager *RedisManager) *LogStreamer {
	glog.Info("LogStreamer-svc Instantiated")
	return &LogStreamer{
		redisManager: redisManager,
	}
}

func formatError(err error) *models.APIError {
	if err == nil {
		return nil
	}
	if err.Error() == KeyNotFoundError {
		return &models.APIError{
			StatusCode: &status412,
			Message:    &status412SubscriberMessage,
		}
	}

	return &models.APIError{
		StatusCode: &status500,
		Message:    &status500Message,
	}
}

func (streamer *LogStreamer) GetStreamLogs(subKey, latestTS string) (*models.GetLogsResponse, *models.APIError) {
	glog.V(5).Infof("GetStreamLogs: %s, %s", subKey, latestTS)
	val, err := streamer.redisManager.GetRedisKey(subKey)
	if err != nil {
		glog.Errorf("GetRedisKey failed for key: %s, err: %s", subKey, err)
		return nil, formatError(err)
	}
	// Check if publisher is still alive
	if _, err = streamer.redisManager.GetRedisKey(val.PeerRedisKey); err != nil {
		glog.Errorf("Peerkey fetch failed for key: %s, peerkey: %s err: %s", subKey, val.PeerRedisKey, err)
		return nil, formatError(err)
	}

	// Get data from stream
	if res, latestReadTS, err := streamer.redisManager.GetStreamLogs(val.RedisStreamName, latestTS); err != nil {
		if err.Error() == StreamNoDataError {
			// No data found for the stream so it is not actully an error.
			glog.V(5).Infof("Error in reading stream data Err:%v LatestTS: %s", err, latestReadTS)
			emptyResp := ""
			return &models.GetLogsResponse{
				Logs:            &emptyResp,
				LatestTimeStamp: &latestTS, // return same timestamp client sent!
			}, nil
		}
		return nil, formatError(err) // 500 error
	} else {
		// Increase expiry time for subscriber key
		if err := streamer.redisManager.SetRedisKeyExpiry(subKey); err != nil {
			return nil, formatError(err)
		}

		glog.V(4).Infof("Got stream data. LatestTS: %s", latestTS)
		return &models.GetLogsResponse{
			Logs:            &res,
			LatestTimeStamp: &latestReadTS,
		}, nil
	}
}

func (streamer *LogStreamer) PutStreamLogs(endpoint string, contents string) *models.APIError {
	glog.V(5).Infof("PutStreamLogs: endpoint: %s", endpoint)
	return formatError(streamer.redisManager.PutStreamData(endpoint, contents))
}

func (streamer *LogStreamer) PublisherHeartbeat(key string) *models.APIError {
	glog.V(5).Infof("Publisher heartbeat: endpoint: %s", key)

	if err := streamer.redisManager.RecordHeartbeat(key); err != nil {
		glog.V(5).Infof("Publisher recordheartbeat (key: %s) failed err: %s", key, err)
		return formatError(err)
	}
	return nil
}
