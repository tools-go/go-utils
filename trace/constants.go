package log

const (
	LOG_OK = "ok"
)

const (
	LOG_TIME_FORMAT = "[2006-01-02 15:04:05.000-0700]"
)

const (
	LOGTAG_REQUEST_IN       = "_request_in"
	LOGTAG_REQUEST_OUT      = "_request_out"
	LOGTAG_MYSQL_SUCCESS    = "_mysql_succ"
	LOGTAG_MYSQL_FAILURE    = "_mysql_fail"
	LOGTAG_MONGO_SUCCESS    = "_mongo_succ"
	LOGTAG_MONGO_FAILURE    = "_mongo_fail"
	LOGTAG_REDIS_SUCCESS    = "_redis_succ"
	LOGTAG_REDIS_FAILURE    = "_redis_fail"
	LOGTAG_THRIFT_SUCCESS   = "_thrift_succ"
	LOGTAG_THRIFT_FAILURE   = "_thrift_fail"
	LOGTAG_HTTP_SUCCESS     = "_http_succ"
	LOGTAG_HTTP_FAILURE     = "_http_fail"
	LOGTAG_PROXY_SUCCESS    = "_proxy_succ"
	LOGTAG_PROXY_FAILURE    = "_proxy_fail"
	LOGTAG_ENGINE_OK        = "_engine_succ"
	LOGTAG_ENGINE_ERR       = "_engine_fail"
	LOGTAG_ELASTIC_SUCCESS  = "_elastic_succ"
	LOGTAG_ELASTIC_FAILURE  = "_elastic_fail"
	LOGTAG_ACTIVEMQ_SUCCESS = "_activemq_succ"
	LOGTAG_ACTIVEMQ_FAILURE = "_avtivemq_fail"
)

const (
	LOGKEY_BEGIN = "__TIME__"
	LOGKEY_TAG   = "tag"
)

var TagSuccessFailureRelation = map[string]string{
	LOGTAG_MYSQL_SUCCESS:   LOGTAG_MYSQL_FAILURE,
	LOGTAG_MONGO_SUCCESS:   LOGTAG_MONGO_FAILURE,
	LOGTAG_REDIS_SUCCESS:   LOGTAG_REDIS_FAILURE,
	LOGTAG_THRIFT_SUCCESS:  LOGTAG_THRIFT_FAILURE,
	LOGTAG_HTTP_SUCCESS:    LOGTAG_HTTP_FAILURE,
	LOGTAG_PROXY_SUCCESS:   LOGTAG_PROXY_FAILURE,
	LOGTAG_ENGINE_OK:       LOGTAG_ENGINE_ERR,
	LOGTAG_ELASTIC_SUCCESS: LOGTAG_ELASTIC_FAILURE,
}
