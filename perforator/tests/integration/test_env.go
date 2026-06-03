package integration

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/docker/go-connections/nat"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/library/go/test/yatest"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/agent"
	agentcpo "github.com/yandex/perforator/perforator/agent/collector/pkg/agent/custom_profiling_operation"
	profiler_config "github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	agent_gateway_client "github.com/yandex/perforator/perforator/internal/agent_gateway/client"
	"github.com/yandex/perforator/perforator/internal/agent_gateway/client/storage"
	gatewayserver "github.com/yandex/perforator/perforator/internal/agent_gateway/server"
	"github.com/yandex/perforator/perforator/internal/agent_gateway/server/custom_profiling_operation"
	storage_service "github.com/yandex/perforator/perforator/internal/agent_gateway/server/storage"
	"github.com/yandex/perforator/perforator/internal/asyncfilecache"
	tasks "github.com/yandex/perforator/perforator/internal/asynctask/compound"
	proxyserver "github.com/yandex/perforator/perforator/internal/symbolizer/proxy/server"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/certifi"
	"github.com/yandex/perforator/perforator/pkg/clickhouse"
	"github.com/yandex/perforator/perforator/pkg/linux/perfevent"
	"github.com/yandex/perforator/perforator/pkg/postgres"
	s3client "github.com/yandex/perforator/perforator/pkg/s3"
	"github.com/yandex/perforator/perforator/pkg/storage/binary"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	clustertop "github.com/yandex/perforator/perforator/pkg/storage/cluster_top"
	custom_profiling_operation_storage "github.com/yandex/perforator/perforator/pkg/storage/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/pkg/storage/databases"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope"
	microscope_filter "github.com/yandex/perforator/perforator/pkg/storage/microscope/filter"
	"github.com/yandex/perforator/perforator/pkg/storage/profile"
	clickhouse_meta "github.com/yandex/perforator/perforator/pkg/storage/profile/meta/clickhouse"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	PostgresPort         = 5432
	ClickHousePort       = 8123
	ClickHouseNativePort = 9000
	MinioPort            = 9000
	ClickHouseKeeperPort = 9181

	postgresContainerImage  = "postgres:latest"
	postgresStartupTimeout  = 120 * time.Second
	postgresContainerUser   = "perforator"
	postgresContainerPass   = "perforator"
	postgresContainerDBName = "perforator"
)

type S3Buckets struct {
	ProfileBucket     string
	BinaryBucket      string
	TaskResultsBucket string
	GsymBinaryBucket  string
}

type Config struct {
	ProxyConfig        *proxyserver.Config
	AgentGatewayConfig *gatewayserver.Config
	AgentConfig        *agent.Config
}

type IntegrationTestEnv struct {
	cfg *Config

	l xlog.Logger

	postgresContainer         testcontainers.Container
	clickHouseKeeperContainer testcontainers.Container
	clickHouseContainer       testcontainers.Container
	minioContainer            testcontainers.Container

	testNetwork *testcontainers.DockerNetwork

	Databases *databases.Databases

	ProxyServer        *proxyserver.PerforatorServer
	AgentGatewayServer *gatewayserver.Server
	Agent              *agent.PerforatorAgent
	AgentRegistry      xmetrics.Registry

	ProxyGRPCPort           int
	ProxyHTTPPort           int
	ProxyMetricsPort        int
	AgentGatewayGRPCPort    int
	AgentGatewayMetricsPort int

	S3Buckets S3Buckets

	servicesCtx      context.Context
	servicesCancel   context.CancelFunc
	servicesErrGroup *errgroup.Group
}

func NewIntegrationTestEnv(
	l xlog.Logger,
	cfg *Config,
) *IntegrationTestEnv {
	s := &IntegrationTestEnv{
		l:   l,
		cfg: cfg,
	}
	return s
}

func (s *IntegrationTestEnv) allocatePorts() {
	// We use static ports because this test assumes to be started in QEMU VM.
	// TODO: replace when needed
	s.ProxyGRPCPort = 10000
	s.ProxyHTTPPort = 10001
	s.ProxyMetricsPort = 10100

	s.AgentGatewayGRPCPort = 10002
	s.AgentGatewayMetricsPort = 10003
}

func (s *IntegrationTestEnv) Start(ctx context.Context) error {
	s.allocatePorts()
	s.S3Buckets = S3Buckets{
		ProfileBucket:     "perforator-profile",
		BinaryBucket:      "perforator-binary",
		TaskResultsBucket: "perforator-task-results",
		GsymBinaryBucket:  "perforator-binary-gsym",
	}

	s.l.Info(ctx, "Starting integration test environment")

	// 0. Create Network
	err := s.createNetwork(ctx)
	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	// 1. Start ClickHouse Keeper (Dependency for ClickHouse)
	err = s.startClickHouseKeeper(ctx)
	if err != nil {
		return fmt.Errorf("failed to start clickhouse-keeper container: %w", err)
	}

	// 2. Start ClickHouse
	err = s.startClickHouse(ctx)
	if err != nil {
		return fmt.Errorf("failed to start clickhouse container: %w", err)
	}

	// 3. Start Postgres
	err = s.startPostgres(ctx)
	if err != nil {
		return fmt.Errorf("failed to start postgres container: %w", err)
	}

	// 4. Start MinIO
	err = s.startMinio(ctx)
	if err != nil {
		return fmt.Errorf("failed to start minio container: %w", err)
	}

	// 5. Initialize S3 Buckets
	err = s.initS3Buckets(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize s3 buckets: %w", err)
	}

	// 6. Run Migrations
	err = s.runMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// 7. Connect to Databases
	err = s.connectDatabases(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to databases: %w", err)
	}

	// 8. Start Services
	err = s.startServices(ctx)
	if err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	s.l.Info(ctx, "Started integration test environment")

	return nil
}

func (s *IntegrationTestEnv) Finish(ctx context.Context) error {
	s.l.Info(ctx, "Finishing integration test environment")

	if s.servicesCancel != nil {
		s.servicesCancel()
	}
	if s.servicesErrGroup != nil {
		_ = s.servicesErrGroup.Wait()
	}

	var errs []error
	var err error
	if s.postgresContainer != nil {
		err = s.postgresContainer.Terminate(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if s.clickHouseContainer != nil {
		err = s.clickHouseContainer.Terminate(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if s.clickHouseKeeperContainer != nil {
		err = s.clickHouseKeeperContainer.Terminate(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if s.minioContainer != nil {
		err = s.minioContainer.Terminate(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to terminate containers: %w", errors.Join(errs...))
	}

	s.l.Info(ctx, "Finished integration test environment")

	return nil
}

func (s *IntegrationTestEnv) createNetwork(ctx context.Context) error {
	var err error
	s.testNetwork, err = network.New(ctx, network.WithCheckDuplicate())
	return err
}

func (s *IntegrationTestEnv) startClickHouseKeeper(ctx context.Context) error {
	keeperConfigPath := yatest.SourcePath("perforator/tests/integration/db/clickhouse-keeper/keeper_config.xml")

	port := fmt.Sprintf("%d/tcp", ClickHouseKeeperPort)

	req := testcontainers.ContainerRequest{
		Image:        "clickhouse/clickhouse-keeper:latest",
		ExposedPorts: []string{port},
		WaitingFor:   wait.ForListeningPort(nat.Port(port)),
		Name:         "clickhouse-keeper",
		Networks:     []string{s.testNetwork.Name},
		NetworkAliases: map[string][]string{
			s.testNetwork.Name: {"clickhouse-keeper"},
		},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      keeperConfigPath,
				ContainerFilePath: "/etc/clickhouse-keeper/keeper_config.xml",
				FileMode:          0644,
			},
		},
		Env: map[string]string{
			"CLICKHOUSE_SKIP_ASYNCHRONOUS_METRICS": "1",
		},
	}

	var err error
	s.clickHouseKeeperContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	return err
}

func (s *IntegrationTestEnv) startClickHouse(ctx context.Context) error {
	configPath := yatest.SourcePath("perforator/tests/integration/db/clickhouse/config.d/zookeeper_config.xml")
	macrosPath := yatest.SourcePath("perforator/tests/integration/db/clickhouse/macros/macros.xml")
	loggerPath := yatest.SourcePath("perforator/tests/integration/db/clickhouse/config.d/logger.xml")

	httpPort := fmt.Sprintf("%d/tcp", ClickHousePort)
	nativePort := fmt.Sprintf("%d/tcp", ClickHouseNativePort)

	req := testcontainers.ContainerRequest{
		Image:        "clickhouse/clickhouse-server:latest",
		ExposedPorts: []string{httpPort, nativePort},
		Env: map[string]string{
			"CLICKHOUSE_USER":                      "perforator",
			"CLICKHOUSE_PASSWORD":                  "perforator",
			"CLICKHOUSE_DB":                        "perforator",
			"CLICKHOUSE_SKIP_ASYNCHRONOUS_METRICS": "1",
		},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      configPath,
				ContainerFilePath: "/etc/clickhouse-server/config.d/zookeeper_config.xml",
				FileMode:          0644,
			},
			{
				HostFilePath:      macrosPath,
				ContainerFilePath: "/etc/clickhouse-server/config.d/macros.xml",
				FileMode:          0644,
			},
			{
				HostFilePath:      loggerPath,
				ContainerFilePath: "/etc/clickhouse-server/config.d/logger.xml",
				FileMode:          0644,
			},
		},
		WaitingFor: wait.ForHTTP("/ping").WithPort(nat.Port(httpPort)),
		Networks:   []string{s.testNetwork.Name},
	}
	var err error
	s.clickHouseContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	return err
}

func postgresSQLURL(host string, port nat.Port) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		postgresContainerUser,
		postgresContainerPass,
		host,
		port.Port(),
		postgresContainerDBName,
	)
}

func (s *IntegrationTestEnv) startPostgres(ctx context.Context) error {
	port := fmt.Sprintf("%d/tcp", PostgresPort)
	natPort := nat.Port(port)
	req := testcontainers.ContainerRequest{
		Image:        postgresContainerImage,
		ExposedPorts: []string{port},
		Env: map[string]string{
			"POSTGRES_USER":     postgresContainerUser,
			"POSTGRES_PASSWORD": postgresContainerPass,
			"POSTGRES_DB":       postgresContainerDBName,
		},
		WaitingFor: wait.ForSQL(natPort, "pgx", postgresSQLURL).
			WithStartupTimeout(postgresStartupTimeout).
			WithPollInterval(500 * time.Millisecond),
		Networks: []string{s.testNetwork.Name},
	}
	var err error
	s.postgresContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	return err
}

func (s *IntegrationTestEnv) startMinio(ctx context.Context) error {
	port := fmt.Sprintf("%d/tcp", MinioPort)
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{port},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "perforator",
			"MINIO_ROOT_PASSWORD": "perforator",
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForLog("MinIO Object Storage Server"),
		Networks:   []string{s.testNetwork.Name},
	}
	var err error
	s.minioContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	return err
}

func (s *IntegrationTestEnv) runMigrations(ctx context.Context) error {
	pgHost, err := s.postgresContainer.Host(ctx)
	if err != nil {
		return err
	}
	pgPort, err := s.postgresContainer.MappedPort(ctx, nat.Port(fmt.Sprintf("%d/tcp", PostgresPort)))
	if err != nil {
		return err
	}

	chHost, err := s.clickHouseContainer.Host(ctx)
	if err != nil {
		return err
	}
	chPortMapped, err := s.clickHouseContainer.MappedPort(ctx, nat.Port(fmt.Sprintf("%d/tcp", ClickHouseNativePort)))
	if err != nil {
		return err
	}

	// Migrate Postgres
	err = s.runDBMigration(ctx, "postgres", "perforator", pgHost, pgPort.Port())
	if err != nil {
		return err
	}

	// Migrate Clickhouse
	return s.runDBMigration(ctx, "clickhouse", "perforator", chHost, chPortMapped.Port())
}

func (s *IntegrationTestEnv) runDBMigration(ctx context.Context, dbType, dbName, host, port string) error {
	migrateBinPath, err := yatest.BinaryPath("perforator/cmd/migrate/migrate")
	if err != nil {
		return err
	}
	cmd := exec.Command(migrateBinPath, dbType, "up",
		"--hosts", host,
		"--port", port,
		"--db", dbName,
		"--user", "perforator",
		"--pass", "perforator",
		"--plaintext",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s migration for %s failed: %s", dbType, dbName, string(out))
	}
	s.l.Info(ctx, fmt.Sprintf("%s migration for %s finished", dbType, dbName), log.String("output", string(out)))

	return nil
}

func (s *IntegrationTestEnv) initS3Buckets(ctx context.Context) error {
	host, err := s.minioContainer.Host(ctx)
	if err != nil {
		return err
	}
	port, err := s.minioContainer.MappedPort(ctx, nat.Port(fmt.Sprintf("%d/tcp", MinioPort)))
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s:%s", host, port.Port())

	cfg := aws.NewConfig().
		WithCredentials(credentials.NewStaticCredentials("perforator", "perforator", "")).
		WithEndpoint("http://" + endpoint).
		WithRegion("us-east-1").
		WithDisableSSL(true).
		WithS3ForcePathStyle(true)

	sess, err := session.NewSession(cfg)
	if err != nil {
		return err
	}
	client := s3.New(sess)

	buckets := []string{
		s.S3Buckets.ProfileBucket,
		s.S3Buckets.BinaryBucket,
		s.S3Buckets.TaskResultsBucket,
		s.S3Buckets.GsymBinaryBucket,
	}

	for _, bucket := range buckets {
		_, err := client.CreateBucket(&s3.CreateBucketInput{
			Bucket: aws.String(bucket),
		})
		if err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
		}

		if bucket == s.S3Buckets.TaskResultsBucket {
			// This allows public access to task-results bucket.
			// This is needed for user to download the profiles or flamegraphs.
			policy := fmt.Sprintf(`{
				"Version":"2012-10-17",
				"Statement":
					[
						{
							"Effect": "Allow",
							"Principal": {
								"AWS":["*"]
							},
							"Action":["s3:GetObject"],
							"Resource":["arn:aws:s3:::%s/*"]
						}
					]
				}`,
				bucket,
			)
			_, err = client.PutBucketPolicy(&s3.PutBucketPolicyInput{
				Bucket: aws.String(bucket),
				Policy: aws.String(policy),
			})
			if err != nil {
				return fmt.Errorf("failed to set bucket policy for %s: %w", bucket, err)
			}
		}
	}

	return nil
}

func (s *IntegrationTestEnv) setEnvVars(ctx context.Context) error {
	err := os.Setenv("PERFORATOR_DB_PASSWORD", "perforator")
	if err != nil {
		return fmt.Errorf("failed to set PERFORATOR_DB_PASSWORD: %w", err)
	}
	err = os.Setenv("MINIO_ACCESS_KEY", "perforator")
	if err != nil {
		return fmt.Errorf("failed to set MINIO_ACCESS_KEY: %w", err)
	}
	err = os.Setenv("MINIO_SECRET_KEY", "perforator")
	if err != nil {
		return fmt.Errorf("failed to set MINIO_SECRET_KEY: %w", err)
	}
	return nil
}

func (s *IntegrationTestEnv) connectDatabases(ctx context.Context) error {
	err := s.setEnvVars(ctx)
	if err != nil {
		return err
	}
	s.Databases, err = databases.NewDatabases(ctx, ctx, s.l, s.makeDBConfig(ctx, "perforator"), "integration_test_suite", &nop.Registry{})
	return err
}

func (s *IntegrationTestEnv) makeDBConfig(ctx context.Context, dbName string) *databases.Config {
	pgHost, _ := s.postgresContainer.Host(ctx)
	pgPort, _ := s.postgresContainer.MappedPort(ctx, nat.Port(fmt.Sprintf("%d/tcp", PostgresPort)))

	chHost, _ := s.clickHouseContainer.Host(ctx)
	chPort, _ := s.clickHouseContainer.MappedPort(ctx, nat.Port(fmt.Sprintf("%d/tcp", ClickHouseNativePort)))

	minioHost, _ := s.minioContainer.Host(ctx)
	minioPort, _ := s.minioContainer.MappedPort(ctx, nat.Port(fmt.Sprintf("%d/tcp", MinioPort)))

	return &databases.Config{
		PostgresCluster: &postgres.Config{
			Endpoints: []postgres.Endpoint{{
				Host: pgHost,
				Port: uint16(pgPort.Int()),
			}},
			AuthConfig: postgres.AuthConfig{
				User:        "perforator",
				PasswordEnv: "PERFORATOR_DB_PASSWORD",
			},
			DB:      dbName,
			SSLMode: postgres.SSLModeDisable,
		},
		ClickhouseConfig: &clickhouse.Config{
			Replicas:                    []string{fmt.Sprintf("%s:%d", chHost, chPort.Int())},
			User:                        "perforator",
			PasswordEnvironmentVariable: "PERFORATOR_DB_PASSWORD",
			Database:                    dbName,
			TLS:                         NoClientTLSConfig(),
		},
		S3Config: &s3client.Config{
			Endpoint:           fmt.Sprintf("http://%s:%s", minioHost, minioPort.Port()),
			Region:             "us-east-1",
			AccessKeyEnv:       "MINIO_ACCESS_KEY",
			SecretKeyEnv:       "MINIO_SECRET_KEY",
			ForcePathStyle:     aws.Bool(true),
			InsecureSkipVerify: true,
		},
	}
}

func (s *IntegrationTestEnv) makeStorageBundleConfig(ctx context.Context) *bundle.Config {
	return &bundle.Config{
		DBs: *s.makeDBConfig(ctx, "perforator"),
		BinaryStorage: &binary.Config{
			MetaStorage:  binary.PostgresMetaStorage,
			S3Bucket:     s.S3Buckets.BinaryBucket,
			GSYMS3Bucket: s.S3Buckets.GsymBinaryBucket,
		},
		ProfileStorage: &profile.Config{
			MetaStorage: clickhouse_meta.Config{
				Batching: clickhouse_meta.BatchingConfig{
					Size:     1000,
					Interval: 1 * time.Second,
				},
			},
			S3Bucket: s.S3Buckets.ProfileBucket,
		},
		MicroscopeStorage: ptr.T(microscope.Postgres),
		TaskStorage: &tasks.TasksConfig{
			StorageType: tasks.Postgres,
		},
		CustomProfilingOperationStorage: ptr.T(custom_profiling_operation_storage.Postgres),
		ClusterTopStorage: &clustertop.Config{
			GenerationsStorage: clustertop.Postgres,
			AggregationStorage: clustertop.Clickhouse,
		},
	}
}

func (s *IntegrationTestEnv) startServices(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.servicesCtx = ctx
	s.servicesCancel = cancel
	s.servicesErrGroup, ctx = errgroup.WithContext(ctx)

	// Setup Proxy Server
	err := s.setupProxyServer(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup proxy server: %w", err)
	}

	// Setup Agent Gateway Server
	err = s.setupAgentGatewayServer(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup agent gateway server: %w", err)
	}

	// Setup Agent
	return s.setupAgent(ctx)
}

func (s *IntegrationTestEnv) setupProxyServer(ctx context.Context) error {
	conf := s.cfg.ProxyConfig
	conf.StorageConfig = *s.makeStorageBundleConfig(ctx)
	host, err := s.minioContainer.Host(ctx)
	if err != nil {
		return fmt.Errorf("failed to get minio host: %w", err)
	}
	port, err := s.minioContainer.MappedPort(ctx, nat.Port(fmt.Sprintf("%d/tcp", MinioPort)))
	if err != nil {
		return fmt.Errorf("failed to get minio port: %w", err)
	}

	conf.RenderedProfiles = &proxyserver.RenderedProfiles{
		URLPrefix: fmt.Sprintf("http://%s:%s/%s/", host, port.Port(), s.S3Buckets.TaskResultsBucket),
		S3Bucket:  s.S3Buckets.TaskResultsBucket,
	}
	conf.FeaturesConfig.EnableNewProfileMerger = ptr.Bool(true)
	conf.FillDefault()

	s.ProxyServer, err = proxyserver.NewPerforatorServer(conf, s.l, xmetrics.NewRegistry())
	if err != nil {
		return fmt.Errorf("failed to create proxy server: %w", err)
	}

	s.servicesErrGroup.Go(func() error {
		s.l.Info(ctx, "Starting Proxy Server", log.Int("port", s.ProxyGRPCPort))
		err := s.ProxyServer.Run(ctx, &proxyserver.RunConfig{
			GRPCPort:    uint32(s.ProxyGRPCPort),
			HTTPPort:    uint32(s.ProxyHTTPPort),
			MetricsPort: uint32(s.ProxyMetricsPort),
		})
		s.l.Info(ctx, "Proxy Server exited", log.Error(err))
		return err
	})

	return s.waitForPort(ctx, s.ProxyGRPCPort)
}

func (s *IntegrationTestEnv) setupAgentGatewayServer(ctx context.Context) error {
	conf := s.cfg.AgentGatewayConfig
	conf.Port = uint32(s.AgentGatewayGRPCPort)
	conf.MetricsPort = uint32(s.AgentGatewayMetricsPort)
	conf.StorageConfig = *s.makeStorageBundleConfig(ctx)
	conf.FillDefault()

	var err error
	s.AgentGatewayServer, err = gatewayserver.NewServer(conf, s.l, xmetrics.NewRegistry())
	if err != nil {
		return fmt.Errorf("failed to create gateway server: %w", err)
	}

	s.servicesErrGroup.Go(func() error {
		s.l.Info(ctx, "Starting Agent Gateway Server", log.Int("port", int(conf.Port)))
		err := s.AgentGatewayServer.Run(ctx)
		s.l.Info(ctx, "Agent Gateway Server exited", log.Error(err))
		return err
	})

	return s.waitForPort(ctx, int(conf.Port))
}

func (s *IntegrationTestEnv) setupAgent(ctx context.Context) error {
	conf := s.cfg.AgentConfig
	conf.Profiler.FillDefault()

	agentOpts := []agent.Option{}
	if conf.AgentGateway != nil {
		agentOpts = append(agentOpts, agent.WithAgentGateway(conf.AgentGateway))
	} else {
		agentGatewayConfig := &agent_gateway_client.Config{
			Host: "localhost",
			Port: uint32(s.AgentGatewayGRPCPort),
			TLS:  NoClientTLSConfig(),
			StorageClient: storage.Config{
				ProfileCompression: "zstd_6",
			},
		}
		agentGatewayConfig.FillDefault()
		agentOpts = append(agentOpts, agent.WithAgentGateway(agentGatewayConfig))
	}

	if conf.CPOService != nil {
		agentOpts = append(agentOpts, agent.WithCPOService(conf.CPOService))
	}

	s.AgentRegistry = xmetrics.NewRegistry()

	var err error
	s.Agent, err = agent.NewPerforatorAgent(
		s.l.Logger(),
		s.AgentRegistry,
		&conf.Profiler,
		agentOpts...,
	)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	s.servicesErrGroup.Go(func() error {
		s.l.Info(ctx, "Starting Agent")
		err := s.Agent.Run(ctx)
		s.l.Info(ctx, "Agent exited", log.Error(err))
		return err
	})

	return nil
}

func NoClientTLSConfig() certifi.ClientTLSConfig {
	return certifi.ClientTLSConfig{
		Enabled:            false,
		InsecureSkipVerify: true,
	}
}

func NoServerTLSConfig() certifi.ServerTLSConfig {
	return certifi.ServerTLSConfig{
		Enabled: false,
	}
}

func testEnvConfig() *Config {
	return &Config{
		ProxyConfig: &proxyserver.Config{
			Server: proxyserver.ServerConfig{
				Insecure: true,
			},
			BinaryProvider: proxyserver.BinaryProviderConfig{
				FileCache: &asyncfilecache.Config{
					MaxSize:  "10G",
					MaxItems: 1000000,
					RootPath: "/tmp/proxy_file_cache",
				},
			},
			FeaturesConfig: proxyserver.FeaturesConfig{
				EnableCPOExperimental: ptr.Bool(true),
			},
		},
		AgentGatewayConfig: &gatewayserver.Config{
			TLS: NoServerTLSConfig(),
			StorageServiceConfig: &storage_service.ServiceConfig{
				MicroscopePullerConfig: &microscope_filter.Config{
					PullInterval: 10 * time.Second,
				},
			},
			CustomProfilingOperationServiceConfig: &custom_profiling_operation.ServiceConfig{
				PollInterval: 10 * time.Second,
			},
		},
		AgentConfig: &agent.Config{
			Profiler: profiler_config.Config{
				ProcessDiscovery: profiler_config.ProcessDiscoveryConfig{
					IgnoreUnrelatedProcesses: true,
				},
				Egress: profiler_config.EgressConfig{
					Interval: 5 * time.Second,
				},
				BPF: machine.Config{
					TraceWallTime: ptr.Bool(false),
				},
				PerfEvents: []profiler_config.PerfEventConfig{{
					Type:      perfevent.CPUCycles.Name(),
					Frequency: ptr.Uint64(99),
				}},
				FeatureFlagsConfig: profiler_config.FeatureFlagsConfig{
					EnableSampleParsingBypass: ptr.Bool(true),
					EnablePHP:                 ptr.Bool(true),
				},
			},
			CPOService: &agentcpo.ServiceConfig{
				Host: "localhost",
			},
		},
	}
}

func (s *IntegrationTestEnv) waitForPort(ctx context.Context, port int) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		dialer := &net.Dialer{}
		conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			return conn.Close()
		}

		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-ticker.C:
		}
	}
}
