package container

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	postgresRepo "gaze/internal/application/repository/retriever/postgres"
	sqliteRetrieverRepo "gaze/internal/application/repository/retriever/sqlite"
	"gaze/internal/application/service/retriever"
	"gaze/internal/config"
	"gaze/internal/types"
	"gaze/internal/types/interfaces"
)

// NewEngineFactory returns an EngineFactory function closed over db and cfg.
// Registered in dig and injected into VectorStoreService for dynamic registry
// updates.
func NewEngineFactory(db *gorm.DB, cfg *config.Config) interfaces.EngineFactory {
	return func(ctx context.Context, store types.VectorStore) (interfaces.RetrieveEngineService, error) {
		return createEngineServiceFromStore(ctx, store, db, cfg)
	}
}

// createEngineServiceFromStore creates a RetrieveEngineService from a VectorStore's config.
func createEngineServiceFromStore(
	ctx context.Context,
	store types.VectorStore,
	db *gorm.DB,
	cfg *config.Config,
) (interfaces.RetrieveEngineService, error) {
	if err := validateRuntimeVectorStoreAddresses(store); err != nil {
		return nil, err
	}
	switch store.EngineType {
	case types.PostgresRetrieverEngineType:
		return createPostgresEngine(store, db)
	case types.SQLiteRetrieverEngineType:
		return createSQLiteEngine(store, db)
	default:
		return nil, fmt.Errorf("unsupported engine type: %s", store.EngineType)
	}
}

// validateRuntimeVectorStoreAddresses is the final guard for persisted or
// imported vector-store rows. Create/test handlers validate the same fields at
// the input boundary, but runtime construction must not assume every stored row
// was written through those handlers.
func validateRuntimeVectorStoreAddresses(store types.VectorStore) error {
	switch store.EngineType {
	case types.PostgresRetrieverEngineType, types.SQLiteRetrieverEngineType:
		return nil
	default:
		return fmt.Errorf("vector store engine %q has no SSRF address policy", store.EngineType)
	}
}

func createPostgresEngine(store types.VectorStore, db *gorm.DB) (interfaces.RetrieveEngineService, error) {
	if store.ConnectionConfig.UseDefaultConnection {
		repo := postgresRepo.NewPostgresRetrieveEngineRepository(db)
		return retriever.NewKVHybridRetrieveEngine(repo, types.PostgresRetrieverEngineType), nil
	}
	// Only UseDefaultConnection is supported.
	return nil, fmt.Errorf("custom postgres connections not yet supported; use use_default_connection=true")
}

func createSQLiteEngine(_ types.VectorStore, db *gorm.DB) (interfaces.RetrieveEngineService, error) {
	repo := sqliteRetrieverRepo.NewSQLiteRetrieveEngineRepository(db)
	return retriever.NewKVHybridRetrieveEngine(repo, types.SQLiteRetrieverEngineType), nil
}
