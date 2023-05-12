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

func SetJsonAsString(ctx context.Context, secretId string, value any) error {
	raw, _ := json.Marshal(value)
	return SetString(ctx, secretId, string(raw))
}

func SetString(ctx context.Context, secretId string, value string) error {
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("error accessing aws: %w", err)
	}
	sm := secretsmanager.NewFromConfig(awsConfig)
	_, err = sm.UpdateSecret(ctx, &secretsmanager.UpdateSecretInput{
		SecretId:     aws.String(secretId),
		SecretString: aws.String(value),
	})
	return err
}
