package services

import (
	"back/models"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

var db *dynamodb.Client

func init() {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: "http://localhost:8000",
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID: "dummy", SecretAccessKey: "dummy", SessionToken: "dummy",
			},
		}),
	)
	if err != nil {
		panic(err)
	}

	db = dynamodb.NewFromConfig(cfg)
	ensureTableExists()
}

func ensureTableExists() {
	_, err := db.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
		TableName: aws.String("Conversations"),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("UserID"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("Timestamp"),
				AttributeType: types.ScalarAttributeTypeS, // ISO8601形式で保存
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("UserID"),
				KeyType:       types.KeyTypeHash, // パーティションキー
			},
			{
				AttributeName: aws.String("Timestamp"),
				KeyType:       types.KeyTypeRange, // ソートキー
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	if err != nil {
		fmt.Printf("Table might already exist: %v\n", err)
	}
}

func SaveMessage(userID string, role string, content string) (models.Conversation, error) {
	conversation := models.Conversation{
		ID:        uuid.New().String(),
		UserID:    userID,
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	// デバッグログ
	fmt.Printf("Saving conversation: %+v\n", conversation)

	_, err := db.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String("Conversations"),
		Item: map[string]types.AttributeValue{
			"ID":        &types.AttributeValueMemberS{Value: conversation.ID},
			"UserID":    &types.AttributeValueMemberS{Value: conversation.UserID},
			"Role":      &types.AttributeValueMemberS{Value: conversation.Role},
			"Content":   &types.AttributeValueMemberS{Value: conversation.Content},
			"Timestamp": &types.AttributeValueMemberS{Value: conversation.Timestamp.Format(time.RFC3339)},
		},
	})

	// エラーが発生した場合は空の会話とエラーを返す
	if err != nil {
		return models.Conversation{}, err
	}

	// 保存した内容を返す
	return conversation, nil
}

func GetRecentConversations(userID string, limit int) ([]models.Conversation, error) {
	result, err := db.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String("Conversations"),
		KeyConditionExpression: aws.String("UserID = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
		ScanIndexForward: aws.Bool(false),         // 新しい順にソート
		Limit:            aws.Int32(int32(limit)), // 最大5件を取得
	})
	if err != nil {
		return nil, err
	}

	conversations := make([]models.Conversation, 0)
	for _, item := range result.Items {
		timestamp, _ := time.Parse(time.RFC3339, item["Timestamp"].(*types.AttributeValueMemberS).Value)
		conv := models.Conversation{
			ID:        item["ID"].(*types.AttributeValueMemberS).Value,
			UserID:    item["UserID"].(*types.AttributeValueMemberS).Value,
			Role:      item["Role"].(*types.AttributeValueMemberS).Value,
			Content:   item["Content"].(*types.AttributeValueMemberS).Value,
			Timestamp: timestamp,
		}
		conversations = append(conversations, conv)
	}

	// デバッグログで取得結果を確認
	fmt.Println("Conversations from DynamoDB:")
	for i, conv := range conversations {
		fmt.Printf("%d: %+v\n", i+1, conv)
	}

	return conversations, nil
}

func UpdateMessageFlag(userID, timestamp string, isLiked, isDisliked *bool) error {
	fmt.Printf("Updating message - UserID: %s, Timestamp: %s, IsLiked: %v, IsDisliked: %v\n", userID, timestamp, isLiked, isDisliked)

	// 更新フィールドを構築
	updateExpression := "SET"
	expressionAttributeValues := map[string]types.AttributeValue{}
	expressionAttributeNames := map[string]string{}

	if isLiked != nil {
		updateExpression += " #isLiked = :isLiked,"
		expressionAttributeValues[":isLiked"] = &types.AttributeValueMemberBOOL{Value: *isLiked}
		expressionAttributeNames["#isLiked"] = "isLiked"
	}
	if isDisliked != nil {
		updateExpression += " #isDisliked = :isDisliked,"
		expressionAttributeValues[":isDisliked"] = &types.AttributeValueMemberBOOL{Value: *isDisliked}
		expressionAttributeNames["#isDisliked"] = "isDisliked"
	}

	// 末尾のカンマを削除
	if len(expressionAttributeValues) > 0 {
		updateExpression = updateExpression[:len(updateExpression)-1]
	}

	// デバッグログ: UpdateItemInput を確認
	fmt.Printf("UpdateItemInput: UpdateExpression: %s, ExpressionAttributeValues: %+v, ExpressionAttributeNames: %+v\n", updateExpression, expressionAttributeValues, expressionAttributeNames)

	_, err := db.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName: aws.String("Conversations"),
		Key: map[string]types.AttributeValue{
			"UserID":    &types.AttributeValueMemberS{Value: userID},
			"Timestamp": &types.AttributeValueMemberS{Value: timestamp},
		},
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeValues: expressionAttributeValues,
		ExpressionAttributeNames:  expressionAttributeNames,
		ReturnValues:              types.ReturnValueUpdatedNew,
	})

	if err != nil {
		fmt.Printf("DynamoDB Update Error: %v\n", err)
		return err
	}

	fmt.Println("Update successful")
	return nil
}

func GetAllConversations(userID string) ([]models.Conversation, error) {
	result, err := db.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String("Conversations"),
		KeyConditionExpression: aws.String("UserID = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
		ScanIndexForward: aws.Bool(true), // 古い順に並び替え
	})
	if err != nil {
		return nil, err
	}

	conversations := make([]models.Conversation, 0)
	for _, item := range result.Items {
		timestamp, _ := time.Parse(time.RFC3339, item["Timestamp"].(*types.AttributeValueMemberS).Value)
		conv := models.Conversation{
			ID:        item["ID"].(*types.AttributeValueMemberS).Value,
			UserID:    item["UserID"].(*types.AttributeValueMemberS).Value,
			Role:      item["Role"].(*types.AttributeValueMemberS).Value,
			Content:   item["Content"].(*types.AttributeValueMemberS).Value,
			Timestamp: timestamp,
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

func GetDynamoDBClient() *dynamodb.Client {
    customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
        return aws.Endpoint{
            URL: "http://localhost:8000",
        }, nil
    })

    cfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion("us-east-1"),
        config.WithEndpointResolverWithOptions(customResolver),
        config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
            Value: aws.Credentials{
                AccessKeyID: "dummy", SecretAccessKey: "dummy", SessionToken: "dummy",
            },
        }),
    )
    if err != nil {
        panic(err)
    }

    return dynamodb.NewFromConfig(cfg)
}
