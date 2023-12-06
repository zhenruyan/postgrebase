package rediscache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisCluster(addrs []string) (*redis.ClusterClient, func()) {
	cli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:          addrs,
		RouteByLatency: true,
		//RouteRandomly:         false,
		ReadTimeout:           time.Second * 3,
		WriteTimeout:          time.Second * 3,
		ContextTimeoutEnabled: true,
		PoolFIFO:              true,
		PoolSize:              30,
		PoolTimeout:           time.Second * 30,
		MinIdleConns:          30,
		MaxIdleConns:          30,
		ConnMaxIdleTime:       time.Second * 30,
		ConnMaxLifetime:       time.Second * 30,
	})

	err := cli.ForEachShard(context.Background(), func(ctx context.Context, shard *redis.Client) error {
		return shard.Ping(ctx).Err()
	})
	if err != nil {
		panic(err)
	}

	return cli, func() {
		_ = cli.Close()
	}
}

func NewRedis(network string, addrs string, db int) (*redis.Client, func()) {
	redisClient := redis.NewClient(&redis.Options{
		//连接信息
		Network: network, //网络类型，tcp or unix，默认tcp
		Addr:    addrs,   //主机名+冒号+端口，默认localhost:6379
		DB:      db,      // redis数据库index

		//连接池容量及闲置连接数量
		PoolSize:     32, // 连接池最大socket连接数，默认为4倍CPU数， 4 * runtime.NumCPU
		MinIdleConns: 8,  //在启动阶段创建指定数量的Idle连接，并长期维持idle状态的连接数不少于指定数量；。

		//超时
		DialTimeout:  5 * time.Second, //连接建立超时时间，默认5秒。
		ReadTimeout:  3 * time.Second, //读超时，默认3秒， -1表示取消读超时
		WriteTimeout: 3 * time.Second, //写超时，默认等于读超时
		PoolTimeout:  4 * time.Second, //当所有连接都处在繁忙状态时，客户端等待可用连接的最大等待时长，默认为读超时+1秒。

		//命令执行失败时的重试策略
		MaxRetries:      0,                      // 命令执行失败时，最多重试多少次，默认为0即不重试
		MinRetryBackoff: 8 * time.Millisecond,   //每次计算重试间隔时间的下限，默认8毫秒，-1表示取消间隔
		MaxRetryBackoff: 512 * time.Millisecond, //每次计算重试间隔时间的上限，默认512毫秒，-1表示取消间隔
	})

	cleanup := func() {
		_ = redisClient.Close()
	}
	return redisClient, cleanup
}

func NewRedisRing(addrs map[string]string, db int) (*redis.Ring, func()) {
	cli := redis.NewRing(&redis.RingOptions{
		Addrs: addrs,
		//RouteRandomly:         false,
		ReadTimeout:     time.Second * 3,
		WriteTimeout:    time.Second * 3,
		PoolFIFO:        true,
		PoolSize:        30,
		PoolTimeout:     time.Second * 30,
		MinIdleConns:    30,
		MaxIdleConns:    30,
		ConnMaxIdleTime: time.Second * 30,
		ConnMaxLifetime: time.Second * 30,
		DB:              db,
	})

	err := cli.ForEachShard(context.Background(), func(ctx context.Context, shard *redis.Client) error {
		return shard.Ping(ctx).Err()
	})
	if err != nil {
		panic(err)
	}

	return cli, func() {
		_ = cli.Close()
	}
}

func NewRedisParseUrl(url string) (*redis.Client, func()) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		panic(err)
	}
	if opt.PoolSize < 32 {
		opt.PoolSize = 32
	}
	if opt.MinIdleConns < 8 {
		opt.MinIdleConns = 8
	}
	if opt.DialTimeout < 5*time.Second {
		opt.DialTimeout = 5 * time.Second
	}
	if opt.ReadTimeout < 5*time.Second {
		opt.ReadTimeout = 5 * time.Second
	}
	if opt.WriteTimeout < 5*time.Second {
		opt.WriteTimeout = 5 * time.Second
	}
	if opt.PoolTimeout < 5*time.Second {
		opt.PoolTimeout = 5 * time.Second
	}
	if opt.MaxRetries < 3 {
		opt.MaxRetries = 3
	}
	redisClient := redis.NewClient(opt)
	cleanup := func() {
		_ = redisClient.Close()
	}
	return redisClient, cleanup
}
