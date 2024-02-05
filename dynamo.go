package main

import (
    "fmt"
    "context"
    "errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
    "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func DynamoClient() *dynamodb.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO())
    if err != nil {
        zapLogger.Fatal(fmt.Sprintf("unable to load SDK config", err.Error()))
        return nil
    }
	cfg.RetryMaxAttempts = 30
    return dynamodb.NewFromConfig(cfg)
}

func TableExists(client *dynamodb.Client, table string) bool {
	exists := true
	_, err := client.DescribeTable(
		context.TODO(), &dynamodb.DescribeTableInput{TableName: aws.String(table)},
	)
	if err != nil {
		var notFoundEx *types.ResourceNotFoundException
		if errors.As(err, &notFoundEx) {
			zapLogger.Error(fmt.Sprintf("Table %v does not exist", table))
		} else {
			zapLogger.Error(fmt.Sprintf("Couldn't determine existence of table %v. Here's why: %v", table, err))
		}
		exists = false
	}
	return exists
}

func AddURLToTable(client *dynamodb.Client, table string, url URLDocument) error {
	item, err := attributevalue.MarshalMap(url)
	if err != nil {
		panic(err)
	}
	_, err = client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(table), Item: item,
	})
	if err != nil {
		zapLogger.Error(fmt.Sprintf("Couldn't add item to table. Here's why: %v", err))
	}
	return err
}

func DeleteURLFromTable(client *dynamodb.Client, table string, url URLDocument) error {
	id, err := url.GetKey()
	if err != nil {
		return err
	}
	_, err = client.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String(table), Key: id,
	})
	if err != nil {
		zapLogger.Error(fmt.Sprintf("Couldn't delete %v from the table. Here's why: %v", url.ID, err))
	}
	return err
}

func GetURLFromTable(ctx context.Context, client *dynamodb.Client, table string, urlid string) (URLDocument, error) {
	url := URLDocument{ID: urlid}
	id, err := url.GetKey()
	if err != nil {
		return url, err
	}
	response, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		Key: id, TableName: aws.String(table),
	})
	if err != nil {
		zapLogger.Error(fmt.Sprintf("Couldn't get info about %v. Here's why: %v", id, err))
	} else {
		err = attributevalue.UnmarshalMap(response.Item, &url)
		if err != nil {
			zapLogger.Error(fmt.Sprintf("Couldn't unmarshal response. Here's why: %v", err))
		}
	}
	return url, err
}