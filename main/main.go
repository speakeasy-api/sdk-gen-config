package main

import "github.com/speakeasy-api/sdk-gen-config/workflow"

func main() {
	res := workflow.GetFileStatus("https://petstore3.swagger.io/api/v3/openapi.yaml")
	println(res)
}
