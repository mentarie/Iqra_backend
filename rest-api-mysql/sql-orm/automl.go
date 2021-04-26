package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"

	automl "cloud.google.com/go/automl/apiv1"
	automlpb "google.golang.org/genproto/googleapis/cloud/automl/v1"
)

// visionClassificationPredict does a prediction for image classification.
func VisionClassificationPredict(final_file_path string) (error, error) {
	projectID := "analog-delight-311114"
	location := "us-central1"
	modelID := "ICN7289082593968914432"
	filePath := final_file_path

	ctx := context.Background()
	client, err := automl.NewPredictionClient(ctx)
	if err != nil {
		return fmt.Errorf("NewPredictionClient: %v", err), nil
	}
	defer client.Close()

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Open: %v", err), nil
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("ReadAll: %v", err), nil
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
		return fmt.Errorf("Predict: %v", err), nil
	}

	for _, payload := range resp.GetPayload() {
		fmt.Printf("Predicted class name: %v\n", payload.GetDisplayName())
		//fmt.Printf("Predicted class score: %v\n", payload.GetClassification().GetScore())
	}

	sort.Slice(resp.GetPayload(), func(i, j int) bool {
		a := resp.GetPayload()[i].Detail
		b := resp.GetPayload()[j].Detail
		log.Println(a)
		log.Println(b)


		return false
	})

	//sort.Slice(a, func(i, j int) bool {
	//	return a[i] > a[j]
	//})
	//for _, v := range a{
	//	fmt.Println(v)
	//}

	return nil, nil
}