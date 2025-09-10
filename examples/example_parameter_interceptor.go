package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-juicedev/juice"
	"github.com/go-juicedev/juice/eval"
)

// ExampleParameterInterceptor demonstrates a simple parameter interceptor
type ExampleParameterInterceptor struct {
	prefix string
}

// Intercept implements the ParameterInterceptor interface
func (e *ExampleParameterInterceptor) Intercept(ctx context.Context, param juice.Param) (juice.Param, error) {
	log.Printf("Intercepting parameter: %v", param)
	
	// Add a prefix to the parameter if it's a map
	if paramMap, ok := param.(map[string]interface{}); ok {
		newParamMap := make(map[string]interface{})
		for k, v := range paramMap {
			newParamMap[k] = v
		}
		newParamMap["tablePrefix"] = e.prefix
		log.Printf("Modified parameter with prefix: %s", e.prefix)
		return newParamMap, nil
	}
	
	return param, nil
}

func main() {
	// Create a simple interceptor
	interceptor := &ExampleParameterInterceptor{prefix: "prod_"}

	// Create a group of interceptors
	interceptors := juice.ParameterInterceptorGroup{interceptor}

	// Test with a parameter
	originalParam := map[string]interface{}{
		"id":   1,
		"name": "test",
	}

	// Apply interceptors
	ctx := context.Background()
	resultParam, err := interceptors.Intercept(ctx, originalParam)
	if err != nil {
		log.Fatalf("Error intercepting parameter: %v", err)
	}

	// Display results
	fmt.Printf("Original parameter: %v\n", originalParam)
	fmt.Printf("Modified parameter: %v\n", resultParam)

	// If the result is a map, show the added field
	if resultMap, ok := resultParam.(map[string]interface{}); ok {
		if prefix, exists := resultMap["tablePrefix"]; exists {
			fmt.Printf("Added table prefix: %v\n", prefix)
		}
	}
}