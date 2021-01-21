package devtoolsservice

var (
	status200                  = int16(200)
	status412                  = int16(412)
	status412PublisherMessage  = "Client may have been stopped reading logs" // Send to publisher
	status412SubscriberMessage = "Container may have stopped streaming logs" // Send to clients

	status500        = int16(500)
	status500Message = "internal server error"
)
