package config

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	BASE_CLOUD_METADATA_URL = "http://169.254.169.254/"
)

var (
	cloudInfo map[string]string
)

func CollectCloudMetrics() map[string]string {

	cloudInfo = make(map[string]string)
	startTime := time.Now()

	// check if base url is accessible, if yes, then continue other cloud check
	_, ok := callUrl("GET", BASE_CLOUD_METADATA_URL, nil)
	if !ok {
		log.Debug("Base URL is not accessible / timedout, hence not accessing other cloud to fetch region, zone, location etc.,")
		return cloudInfo
	}

	getAwsCloudDetails()
	getGoogleCloudDetails()
	getAzureCloudDetails()
	totalTimeTaken := time.Since(startTime)
	log.Debug("Total time taken to get Cloud params ", totalTimeTaken)

	return cloudInfo
}

func getAwsCloudDetails() {
	var token, awsRegion, awsAvailZoneId, awsAvailZone string
	var tokenHeaders = make(map[string]string)
	tokenHeaders["X-aws-ec2-metadata-token-ttl-seconds"] = "21600"
	token, ok := callUrl("PUT", BASE_CLOUD_METADATA_URL+"/latest/api/token", tokenHeaders)
	if !ok {
		return
	}

	tokenHeaders["X-aws-ec2-metadata-token"] = token
	awsRegion, ok = callUrl("GET", BASE_CLOUD_METADATA_URL+"/latest/meta-data/placement/region", tokenHeaders)
	if !ok {
		return
	}

	awsAvailZoneId, ok = callUrl("GET", BASE_CLOUD_METADATA_URL+"/latest/meta-data/placement/availability-zone-id", tokenHeaders)
	if !ok {
		return
	}
	awsAvailZone, ok = callUrl("GET", BASE_CLOUD_METADATA_URL+"/latest/meta-data/placement/availability-zone", tokenHeaders)
	if !ok {
		return
	}

	cloudInfo["aws_region"] = awsRegion
	cloudInfo["aws_availability_zone_id"] = awsAvailZoneId
	cloudInfo["aws_availability_zone"] = awsAvailZone
}

func getAzureCloudDetails() {
	var tokenHeaders = make(map[string]string)
	tokenHeaders["Metadata"] = "true"
	azureLocation, ok := callUrl("GET", BASE_CLOUD_METADATA_URL+"/metadata/instance/compute/location?api-version=2021-02-01&format=text", tokenHeaders)
	if !ok {
		return
	}

	azureZone, ok := callUrl("GET", BASE_CLOUD_METADATA_URL+"/metadata/instance/compute/zone?api-version=2021-02-01&format=text", tokenHeaders)
	if !ok {
		return
	}

	cloudInfo["azure_location"] = azureLocation
	cloudInfo["azure_zone"] = azureZone

}

func getGoogleCloudDetails() {
	var tokenHeaders = make(map[string]string)
	tokenHeaders["Metadata-Flavor"] = "Google"
	gcpZone, ok := callUrl("GET", BASE_CLOUD_METADATA_URL+"/computeMetadata/v1/instance/zone", tokenHeaders)
	if !ok {
		return
	}

	cloudInfo["gcp_zone"] = gcpZone

}

func callUrl(method string, url string, headers map[string]string) (string, bool) {
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		log.Debug("Error while creating new-http-request, Error ", err)
		return "", false
	}

	request.Header.Set("Content-Type", "text/html; charset=utf-8")
	if headers != nil {
		for k, v := range headers {
			request.Header.Set(k, v)
		}
	}

	// send the request
	client := &http.Client{}
	response, err := client.Do(request)

	if err != nil {
		log.Debug("Call failed to URL ", url, " Error ", err)
		return "", false
	}

	esponseBodyBytes, err := io.ReadAll(response.Body)

	if err != nil {
		fmt.Println("Error 2 ", err)
		return "", false
	}

	responseBody := string(esponseBodyBytes)

	if strings.Contains(responseBody, "xml version=") {
		return "", false
	}

	return responseBody, true
}
