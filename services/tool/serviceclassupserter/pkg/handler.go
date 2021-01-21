package pkg

// This contains the main handler code
import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	metafile = "metadata.json"
)

// ServiceClassMetadata keeps the metadata of the Service Class files
type ServiceClassMetadata struct {
	// Includes keep the list of all the files to be included in the create/update
	Includes []string `json:"includes"`
	// Exclude the given types from deletion when delete missing is set
	DeleteMissingExcludes []string `json:"deleteMissingExcludes"`
}

// UpsertServiceClasses is the entry point for starting the Server Class upserter
func UpsertServiceClasses(ctx context.Context,
	queryParam *model.ServiceClassQueryParam,
	createHandler func(context.Context, api.ObjectModelAPI, map[string]*model.ServiceClass) error,
	updateHandler func(context.Context, api.ObjectModelAPI, map[string]*model.ServiceClass) error,
	deleteHandler func(context.Context, api.ObjectModelAPI, map[string]bool) error) error {

	dbAPI, err := api.NewObjectModelAPI()
	if err != nil {
		return err
	}
	defer dbAPI.Close()
	dataDir := path.Clean(Cfg.DataDir)
	if !strings.HasSuffix(dataDir, "/") {
		dataDir = dataDir + "/"
	}
	metafile := filepath.Join(dataDir, metafile)
	file, err := os.Open(metafile)
	if err != nil {
		return err
	}
	defer file.Close()
	metadata := &ServiceClassMetadata{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(metadata)
	if err != nil {
		return err
	}
	includeFiles := map[string]bool{}
	for _, relPath := range metadata.Includes {
		relPath = path.Clean(relPath)
		if _, ok := includeFiles[relPath]; ok {
			return fmt.Errorf("Duplicate file path found %s", relPath)
		}
		includeFiles[relPath] = true
	}
	deleteMissingExcludes := map[string]bool{}
	for _, svcType := range metadata.DeleteMissingExcludes {
		if deleteMissingExcludes[svcType] {
			return fmt.Errorf("Duplicate type %s", svcType)
		}
		deleteMissingExcludes[svcType] = true
	}
	addSvcClasses := map[string]*model.ServiceClass{}
	updateSvcClasses := map[string]*model.ServiceClass{}
	deleteSvcClassIDs := map[string]bool{}
	err = filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if path == metafile {
			return nil
		}
		filename := info.Name()
		if filepath.Ext(filename) != ".json" {
			fmt.Printf("Ignoring file %s\n", filename)
			return nil
		}
		relPath := strings.TrimPrefix(path, dataDir)
		if !includeFiles[relPath] {
			fmt.Printf("Skipping file %s\n", relPath)
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			fmt.Printf("Error opening file %s. Error: %s\n", path, err.Error())
			return err
		}
		defer file.Close()
		fmt.Printf("Processing file %s\n", relPath)
		svcClass := model.ServiceClass{}
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&svcClass)
		if err != nil {
			fmt.Printf("Error occurred in decoding Service Class at %s. Error: %s\n", relPath, err.Error())
			return err
		}
		addSvcClasses[svcClass.Type] = &svcClass
		return nil
	})
	if err != nil {
		fmt.Printf("Error occurred in traversing data directory %s. Error: %s\n", dataDir, err.Error())
		return err
	}
	entitiesQueryParam := &model.EntitiesQueryParam{}
	if queryParam == nil {
		queryParam = &model.ServiceClassQueryParam{}
	}
	fmt.Print("Fetching existing Service Classes...\n")
	selectResponse, err := dbAPI.SelectAllServiceClasses(ctx, entitiesQueryParam, queryParam)
	if err != nil {
		return err
	}
	for _, svcClass := range selectResponse.SvcClassList {
		inSvcClass, ok := addSvcClasses[svcClass.Type]
		if !ok {
			if Cfg.DeleteOnMissing && !deleteMissingExcludes[svcClass.Type] {
				deleteSvcClassIDs[svcClass.Type] = true
			} else {
				fmt.Printf("Skipping deletion of Service Class %+v\n", svcClass)
			}
			continue
		}
		// Set ID from the existing Service Class
		inSvcClass.ID = svcClass.ID
		delete(addSvcClasses, svcClass.Type)
		if IsServiceClassUpdated(&svcClass, inSvcClass) {
			updateSvcClasses[svcClass.Type] = inSvcClass
		} else {
			fmt.Printf("Unmodified Service Class %+v\n", svcClass)
		}
	}
	// Apply delete first to clean all if some private namespace wants to start from scratch.
	// Name collision and other constraints can be avoided
	if len(deleteSvcClassIDs) > 0 {
		err = deleteHandler(ctx, dbAPI, deleteSvcClassIDs)
		if err != nil {
			return err
		}
	}
	if len(addSvcClasses) > 0 {
		err = createHandler(ctx, dbAPI, addSvcClasses)
		if err != nil {
			return err
		}
	}
	if len(updateSvcClasses) > 0 {
		err = updateHandler(ctx, dbAPI, updateSvcClasses)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateServiceClasses is the handler to create Service Classes
func CreateServiceClasses(ctx context.Context, dbAPI api.ObjectModelAPI, svcClasses map[string]*model.ServiceClass) error {
	fmt.Print("\nCreating Service Classes...\n")
	for _, svcClass := range svcClasses {
		fmt.Printf("Creating Service Class %+v\n", svcClass)
		if !Cfg.DisableDryRun {
			// Keeping this check inside to print the message above
			continue
		}
		_, err := dbAPI.CreateServiceClass(ctx, svcClass, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateServiceClasses is the handler to update Service Classes
func UpdateServiceClasses(ctx context.Context, dbAPI api.ObjectModelAPI, svcClasses map[string]*model.ServiceClass) error {
	fmt.Print("\nUpdating Service Classes...\n")
	for _, svcClass := range svcClasses {
		fmt.Printf("Updating Service Class %+v\n", svcClass)
		if !Cfg.DisableDryRun {
			// Keeping this check inside to print the message above
			continue
		}
		_, err := dbAPI.UpdateServiceClass(ctx, svcClass, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteServiceClasses is the handler to delete Service Classes
func DeleteServiceClasses(ctx context.Context, dbAPI api.ObjectModelAPI, svcClassIDs map[string]bool) error {
	fmt.Print("\nDeleting Service Classes...\n")
	for svcClassID := range svcClassIDs {
		fmt.Printf("Deleting Service Class %s\n", svcClassID)
		if !Cfg.DisableDryRun {
			// Keeping this check inside to print the message above
			continue
		}
		_, err := dbAPI.DeleteServiceClass(ctx, svcClassID, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// IsServiceClassUpdated checks if a Service Class has been updated
func IsServiceClassUpdated(oldSvcClass, newSvcClass *model.ServiceClass) bool {
	oldSvcClass.ID = newSvcClass.ID
	oldSvcClass.Version = newSvcClass.Version
	oldSvcClass.CreatedAt = newSvcClass.CreatedAt
	oldSvcClass.UpdatedAt = newSvcClass.UpdatedAt
	return !reflect.DeepEqual(oldSvcClass, newSvcClass)
}
