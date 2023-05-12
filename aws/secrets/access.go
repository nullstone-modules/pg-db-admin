package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func GetString(ctx context.Context, secretId string) (string, error) {
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("error accessing aws: %w", err)
	}
	sm := secretsmanager.NewFromConfig(awsConfig)
	out, err := sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(secretId)})
	if err != nil {
		return "", fmt.Errorf("error retrieving secret (%s): %w", secretId, err)
	}
	if out.SecretString == nil {
		return "", nil
	}
	return *out.SecretString, nil
}

func GetLatestVersionId(ctx context.Context, secretId string) (string, error) {
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("error accessing aws: %w", err)
	}
	sm := secretsmanager.NewFromConfig(awsConfig)
	out, err := sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(secretId)})
	if err != nil {
		return "", fmt.Errorf("error retrieving secret version id (%s): %w", secretId, err)
	}
	if out.VersionId == nil {
		return "", nil
	}
	return *out.VersionId, nil
}

func SetJsonAsString(ctx context.Context, secretId string, value any) (string, error) {
	raw, _ := json.Marshal(value)
	return SetString(ctx, secretId, string(raw))
}

// SetString creates a secret version containing the input value
// The new VersionId is returned
func SetString(ctx context.Context, secretId string, value string) (string, error) {
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("error accessing aws: %w", err)
	}
	sm := secretsmanager.NewFromConfig(awsConfig)
	input := &secretsmanager.UpdateSecretInput{
		SecretId:     aws.String(secretId),
		SecretString: aws.String(value),
	}
	if out, err := sm.UpdateSecret(ctx, input); err != nil {
		return "", fmt.Errorf("unable to update secret (%s) value: %w", secretId, err)
	} else if out.VersionId != nil {
		return *out.VersionId, nil
	}
	return "", nil
}
