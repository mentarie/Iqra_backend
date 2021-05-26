package main

import (
	automl "cloud.google.com/go/automl/apiv1"
	automlpb "google.golang.org/genproto/googleapis/cloud/automl/v1"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
)

// visionClassificationPredict does a prediction for image classification.
func VisionClassificationPredict(final_file_path string) (float32, string, error) {
	projectID := "stoked-cirrus-314800"
	location := "us-central1"
	modelID := "ICN519398297845104640"
	//final_file_path = /Users/mac/go/src/github.com/mentarie/Iqra_backend/rest-api-mysql/sql-orm/
	filePath := final_file_path

	ctx := context.Background()
	client, err := automl.NewPredictionClient(ctx)
	if err != nil {
		return 0, "", fmt.Errorf("NewPredictionClient: %v", err)
	}
	defer client.Close()

	file, err := os.Open(filePath)
	if err != nil {
		return 0, "", fmt.Errorf("Open: %v", err)
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return 0, "", fmt.Errorf("ReadAll: %v", err)
	}

	req := &automlpb.PredictRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/models/%s", projectID, location, modelID),
		Payload: &automlpb.ExamplePayload{
			Payload: &automlpb.ExamplePayload_Image{
				Image: &automlpb.Image{
					Data: &automlpb.Image_ImageBytes{
						ImageBytes: bytes,
					},
				},
			},
		},
		// Params is additional domain-specific parameters.
		Params: map[string]string{
			// score_threshold is used to filter the result.
			"score_threshold": "0.0",
		},
	}

	resp, err := client.Predict(ctx, req)
	if err != nil {
		return 0, "", fmt.Errorf("Predict: %v", err)
	}

	for _, payload := range resp.GetPayload() {
		fmt.Printf("Predicted class name: %v\n", payload.GetDisplayName())
		fmt.Printf("Predicted class score: %v\n", payload.GetClassification().GetScore())
	}

	//hasil score
	m := resp.GetPayload()
	sort.Slice(m, func(i, j int) bool {
		return m[i].GetClassification().Score > m[j].GetClassification().Score
	})
	hasil := m[0].GetClassification().Score
	name := m[0].DisplayName

	return hasil, name, nil
}