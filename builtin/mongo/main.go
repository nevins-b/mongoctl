package mongo

type MongoConfig struct {
	Id       string                 `bson:"_id"`
	Version  int64                  `bson:"version"`
	Members  []*MongoClusterMember  `bson:"members"`
	Settings map[string]interface{} `bson:"settings"`
}

type MongoClusterMember struct {
	Id           int64             `bson:"_id"`
	Host         string            `bson:"host"`
	ArbiterOnly  bool              `bson:"arbiterOnly"`
	BuildIndexes bool              `bson:"buildIndexes"`
	Hidden       bool              `bson:"hidden"`
	Priority     int64             `bson:"priority"`
	Tags         map[string]string `bson:"tags"`
	SlaveDelay   int64             `bson:"slaveDelay"`
	Votes        int64             `bson:"votes"`
}
