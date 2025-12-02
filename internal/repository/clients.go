package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	rdsutils "github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	_ "github.com/lib/pq"
)

type Config struct {
	Profile      string // Primarily for dev purposes
	S3BucketName string
	Region       string

	DBEndpoint string // e.g. erikkn-test.abc123xyz.eu-central-1.rds.amazonaws.com
	DBUser     string // e.g. "masteruser" or some IAM-enabled user
	DBName     string // e.g. "postgres" or your DB name
	DBPort     int    // e.g. 5432
}

type Clients struct {
	RDS    *RDSClient
	S3     *S3Client
	Config *Config
}

type S3Client struct {
	Client     *s3.Client // The actual S3 client
	BucketName string     // The bucket name (from config)
}

// RDSClient encapsulates the PostgreSQL RDS client (sql.DB) with IAM authentication
type RDSClient struct {
	Client *sql.DB // The actual PostgreSQL database client
}

func (c *Config) LoadAWSConfig() (*aws.Config, error) {
	cfg := aws.Config{}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(c.Region), config.WithSharedConfigProfile(c.Profile))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	return &cfg, nil

}

// NewS3Client creates a new S3 client and stores the bucket name
func NewS3Client(cfg *Config) (*S3Client, error) {
	awsCfg, err := cfg.LoadAWSConfig()
	if err != nil {
		// Handle error appropriately
		return nil, fmt.Errorf("Failed to load AWS config for S3 client: %v", err)
	}

	client := s3.NewFromConfig(*awsCfg)
	return &S3Client{
		Client:     client,
		BucketName: cfg.S3BucketName, // Store the bucket name
	}, nil
}

// NewRDSClient creates and returns a new PostgreSQL RDS client using IAM authentication
func (c *Config) NewRDSClient() (*RDSClient, error) {
	// Step 1: Load AWS config (credentials, region, etc.)
	awsCfg, err := c.LoadAWSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for RDS: %v", err)
	}

	endpointWithPort := fmt.Sprintf("%s:%d", c.DBEndpoint, c.DBPort)

	// This operation is performed locally, not an API call
	authToken, err := rdsutils.BuildAuthToken(
		context.TODO(),
		endpointWithPort,
		c.Region,
		c.DBUser,
		awsCfg.Credentials, // Uses the loaded credentials provider from aws.Config
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create authentication token: %w", err)
	}

	escapedUser := url.QueryEscape(c.DBUser)
	escapedToken := url.QueryEscape(authToken)
	escapedDB := url.QueryEscape(c.DBName)

	// 2. Use the token as the password in a standard database connection string
	// For PostgreSQL (using pgx driver):
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=require",
		escapedUser,
		escapedToken,
		c.DBEndpoint,
		escapedDB,
	)

	// Step 4: Open the PostgreSQL connection (sql.DB)
	db, err := sql.Open("postgres", connStr) // Use "postgres" driver for PostgreSQL
	if err != nil {
		return nil, fmt.Errorf("failed to open DB connection: %v", err)
	}

	// Step 5: Ping the DB to ensure the connection is working
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping RDS PostgreSQL database: %v", err)
	}

	// Return the established database connection wrapped in RDSClient
	return &RDSClient{Client: db}, nil
}

// NewAWSClients creates and returns a new Clients object with RDS and S3 clients
func NewAWSClients(cfg *Config) (*Clients, error) {
	// Create the S3 client
	s3Client, err := NewS3Client(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating S3 client: %v", err)
	}

	// Create the RDS client (PostgreSQL connection)
	rdsClient, err := cfg.NewRDSClient()
	if err != nil {
		return nil, fmt.Errorf("error creating RDS client: %v", err)
	}

	// Return the Clients object, which includes both RDS and S3 clients
	return &Clients{
		RDS: rdsClient,
		//RDS:    nil,
		S3:     s3Client,
		Config: cfg,
	}, nil
}
