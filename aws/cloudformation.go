package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

var (
	numberOfRetries = 10
	throttleDelay   = client.DefaultRetryerMinRetryDelay
)

var cfClient *cloudformation.CloudFormation

type Cloudformation struct {
	client *cloudformation.CloudFormation
}

func (c *Cloudformation) GetOutput(stackname string) (map[string]string, error) {
	result := map[string]string{}

	resp, _ := c.client.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackname),
	})

	if len(resp.Stacks) <= 0 {
		return nil, fmt.Errorf("%s stack does not exist", stackname)
	}

	stack := resp.Stacks[0]

	for _, output := range stack.Outputs {
		result[*output.OutputKey] = *output.OutputValue
	}

	return result, nil
}

func (c *Cloudformation) GetOutputs(stacknames []string) (map[string]string, error) {
	result := map[string]string{}

	for _, stackname := range stacknames {
		outputs, err := c.GetOutput(stackname)

		if err != nil {
			continue
		}

		for key, value := range outputs {
			result[key] = value
		}
	}

	return result, nil
}

func NewCloudformation() Cloudformation {
	if cfClient == nil {
		retryer := client.DefaultRetryer{
			NumMaxRetries:    numberOfRetries,
			MinThrottleDelay: throttleDelay,
		}

		cfClient = cloudformation.New(Session, &aws.Config{
			Retryer: retryer,
		})
	}

	return Cloudformation{client: cfClient}
}
