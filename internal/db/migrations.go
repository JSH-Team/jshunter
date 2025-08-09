package db

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func RegisterEndpointsCollection(app core.App, jsFilesCollection *core.Collection) (*core.Collection, error) {
	endpointCollection := core.NewBaseCollection("endpoints")

	endpointCollection.Fields.Add(
		&core.TextField{
			Name:     "url",
			Required: false,
			Max:      50000,
		},
		&core.TextField{
			Name:     "query_string",
			Required: false,
			Max:      50000,
		},
		&core.JSONField{
			Name:     "request_headers",
			Required: false,
			Hidden:   true,
		},
		&core.TextField{
			Name:     "hash",
			Required: false,
			Max:      256,
		},
		&core.TextField{
			Name:     "mobile_hash",
			Required: false,
			Max:      256,
		},
		&core.SelectField{
			Name:     "extraction_status",
			Required: false,
			Values:   []string{"pending", "processing", "processed", "failed"},
			Hidden:   true,
		},
		&core.SelectField{
			Name:     "prettify_status",
			Required: false,
			Values:   []string{"pending", "processing", "processed", "failed"},
			Hidden:   true,
		},
		&core.DateField{
			Name:     "created_at",
			Required: false,
		},
		&core.RelationField{
			Name:         "js_files",
			Required:     false,
			CollectionId: jsFilesCollection.Id,
			MaxSelect:    10000,
		},
	)

	rule := "id != ''"
	endpointCollection.ViewRule = &rule
	endpointCollection.ListRule = &rule
	endpointCollection.CreateRule = &rule

	if err := app.Save(endpointCollection); err != nil {
		return nil, err
	}

	return endpointCollection, nil
}

func RegisterTempEndpointsCollection(app core.App) (*core.Collection, error) {
	tmpEndpointCollection := core.NewBaseCollection("tmp_endpoints")

	tmpEndpointCollection.Fields.Add(
		&core.TextField{
			Name:     "url",
			Required: false,
			Max:      50000,
		},
		&core.TextField{
			Name:     "query_string",
			Required: false,
			Max:      50000,
		},
		&core.JSONField{
			Name:     "request_headers",
			Required: false,
			Hidden:   true,
		},

		&core.FileField{
			Name:     "tmp_body",
			Required: false,
			MaxSize:  1024 * 1024 * 500,
		},
	)

	rule := "id != ''"
	tmpEndpointCollection.CreateRule = &rule

	if err := app.Save(tmpEndpointCollection); err != nil {
		return nil, err
	}

	return tmpEndpointCollection, nil
}

func RegisterJavaScriptFilesCollection(app core.App) (*core.Collection, error) {
	jsFilesCollection := core.NewBaseCollection("js_files")

	jsFilesCollection.Fields.Add(
		&core.TextField{
			Name:     "url",
			Required: true,
			Max:      50000,
		},
		&core.TextField{
			Name:     "hash",
			Required: false,
			Max:      256,
		},
		&core.TextField{
			Name:     "parent_id",
			Required: false,
		},
		&core.BoolField{
			Name:     "has_chunks",
			Required: false,
		},
		&core.DateField{
			Name:     "created_at",
			Required: false,
		},
		&core.NumberField{
			Name:     "line_count",
			Required: false,
		},
		&core.SelectField{
			Name:     "type",
			Required: true,
			Values:   []string{"normal", "inline", "mobile", "chunk"},
		},
		&core.SelectField{
			Name:     "dechunker_status",
			Required: false,
			Values:   []string{"pending", "processing", "processed", "failed"},
			Hidden:   true,
		},
		&core.SelectField{
			Name:     "prettify_status",
			Required: false,
			Values:   []string{"pending", "processing", "processed", "failed"},
			Hidden:   true,
		},
		&core.SelectField{
			Name:     "analysis_status",
			Required: false,
			Values:   []string{"pending", "processing", "processed", "failed"},
			Hidden:   true,
		},
		&core.SelectField{
			Name:     "sourcemap_status",
			Required: false,
			Values:   []string{"pending", "processing", "processed", "failed"},
			Hidden:   true,
		},
	)

	rule := "id != ''"
	jsFilesCollection.ViewRule = &rule
	jsFilesCollection.ListRule = &rule

	if err := app.Save(jsFilesCollection); err != nil {
		return nil, err
	}

	return jsFilesCollection, nil
}

func RegisterFindingsCollection(app core.App, jsFilesCollection *core.Collection) (*core.Collection, error) {
	findingsCollection := core.NewBaseCollection("findings")

	findingsCollection.Fields.Add(
		&core.TextField{
			Name:     "type",
			Required: true,
		},
		&core.NumberField{
			Name:     "line",
			Required: true,
		},
		&core.NumberField{
			Name:     "column",
			Required: true,
		},
		&core.TextField{
			Name:     "value",
			Required: true,
			Max:      50000,
		},
		&core.JSONField{
			Name:     "metadata",
			Required: false,
			MaxSize:  1024 * 1024 * 100,
		},
		&core.DateField{
			Name:     "created_at",
			Required: true,
		},
		&core.RelationField{
			Name:         "js_file",
			Required:     false,
			CollectionId: jsFilesCollection.Id,
		},
	)

	rule := "id != ''"
	findingsCollection.ListRule = &rule
	findingsCollection.ViewRule = &rule

	return findingsCollection, app.Save(findingsCollection)
}

func init() {
	m.Register(
		// Up migration
		func(app core.App) error {

			jsFilesCollection, err := RegisterJavaScriptFilesCollection(app)
			if err != nil {
				return err
			}

			_, err = RegisterEndpointsCollection(app, jsFilesCollection)
			if err != nil {
				return err
			}

			_, err = RegisterTempEndpointsCollection(app)
			if err != nil {
				return err
			}

			_, err = RegisterFindingsCollection(app, jsFilesCollection)
			if err != nil {
				return err
			}

			return nil
		},

		// Down migration
		func(app core.App) error {
			endpoints, err := app.FindCollectionByNameOrId("endpoints")
			if err == nil {
				if err := app.Delete(endpoints); err != nil {
					return err
				}
			}

			jsFiles, err := app.FindCollectionByNameOrId("js_files")
			if err == nil {
				if err := app.Delete(jsFiles); err != nil {
					return err
				}
			}

			// Findings
			findings, err := app.FindCollectionByNameOrId("findings")
			if err == nil {
				if err := app.Delete(findings); err != nil {
					return err
				}
			}

			return nil
		}, "")

	// Migration to add missing fields to existing collections
	m.Register(
		// Up migration - Add missing fields
		func(app core.App) error {

			return nil
		},

		// Down migration - Remove added fields
		func(app core.App) error {
			// This migration is safe to leave empty as removing fields
			// could cause data loss, better to leave them
			return nil
		}, "")

}
