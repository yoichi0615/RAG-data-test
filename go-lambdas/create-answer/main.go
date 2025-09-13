package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type InquiryEvent struct {
	ID string `json:"id"`
}

type InquiryItem struct {
	ID         string `dynamodbav:"id"`
	ReviewText string `dynamodbav:"reviewText"`
	Category   string `dynamodbav:"category"`
}

var (
	dynamoDBClient *dynamodb.Client
	bedrockClient  *bedrockagentruntime.Client
	inquiryTable   string
	kbID           string
	awsRegion      string
)

func init() {
	inquiryTable = os.Getenv("INQUIRY_TABLE_NAME")
	kbID = os.Getenv("BEDROCK_KNOWLEDGE_BASE_ID")
	awsRegion = os.Getenv("AWS_REGION")

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion))
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}

	dynamoDBClient = dynamodb.NewFromConfig(cfg)
	bedrockClient = bedrockagentruntime.NewFromConfig(cfg)
}

func generateAnswerWithBedrockKB(ctx context.Context, question string, category string) (string, error) {
	var systemPrompt string
	switch category {
	case "ポジティブな感想":
		systemPrompt = "お客様からの嬉しいお言葉に対して、感謝の気持ちを込めて丁寧に返答してください。"
	case "ネガティブな感想":
		systemPrompt = "お客様のご不満に対して、謝罪の気持ちを込めて改善への取り組みを示しながら丁寧に返答してください。"
	case "質問":
		systemPrompt = "お客様からの質問に対して、正確で分かりやすい情報を提供してください。"
	case "改善要望":
		systemPrompt = "お客様からの貴重なご意見として受け止め、検討することをお伝えしながら丁寧に返答してください。"
	default:
		systemPrompt = "お客様からの問い合わせに丁寧で親切な回答をしてください。"
	}

	modelArn := fmt.Sprintf("arn:aws:bedrock:%s::foundation-model/anthropic.claude-3-haiku-20240307-v1:0", awsRegion)
	inputText := fmt.Sprintf("%s\n\n質問: %s", systemPrompt, question)

	output, err := bedrockClient.RetrieveAndGenerate(ctx, &bedrockagentruntime.RetrieveAndGenerateInput{
		Input: &types.RetrieveAndGenerateInput{
			Text: &inputText,
		},
		RetrieveAndGenerateConfiguration: &types.RetrieveAndGenerateConfiguration{
			Type: types.RetrieveAndGenerateTypeKnowledgeBase,
			KnowledgeBaseConfiguration: &types.KnowledgeBaseRetrieveAndGenerateConfiguration{
				KnowledgeBaseId: &kbID,
				ModelArn:        &modelArn,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to retrieve and generate from bedrock: %w", err)
	}

	if output.Output == nil {
		return "", fmt.Errorf("received nil output from bedrock")
	}

	if output.Output != nil && output.Output.Text != nil {
		return *output.Output.Text, nil
	}

	return "", fmt.Errorf("unexpected or empty output from bedrock")
}

func handler(ctx context.Context, event InquiryEvent) (map[string]interface{}, error) {
	if event.ID == "" {
		return nil, fmt.Errorf("missing inquiry id")
	}

	// DynamoDBから問い合わせ内容を取得
	getItemOutput, err := dynamoDBClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &inquiryTable,
		Key: map[string]dynamodbTypes.AttributeValue{
			"id": &dynamodbTypes.AttributeValueMemberS{Value: event.ID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get item from dynamodb: %w", err)
	}
	if getItemOutput.Item == nil {
		return nil, fmt.Errorf("inquiry not found")
	}

	var item InquiryItem
	if err := attributevalue.UnmarshalMap(getItemOutput.Item, &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dynamodb item: %w", err)
	}
	if item.ReviewText == "" {
		return nil, fmt.Errorf("no review text found")
	}

	// BedrockナレッジベースでRAG検索と回答生成
	answer, err := generateAnswerWithBedrockKB(ctx, item.ReviewText, item.Category)
	if err != nil {
		log.Printf("Error generating answer with Bedrock KB: %v", err)
		answer = "申し訳ございませんが、現在回答を生成することができません。後ほど担当者からご連絡いたします。"
	}

	// DynamoDBに回答を保存
	_, err = dynamoDBClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &inquiryTable,
		Key: map[string]dynamodbTypes.AttributeValue{
			"id": &dynamodbTypes.AttributeValueMemberS{Value: event.ID},
		},
		UpdateExpression: aws.String("SET answer = :answer, #status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]dynamodbTypes.AttributeValue{
			":answer": &dynamodbTypes.AttributeValueMemberS{Value: answer},
			":status": &dynamodbTypes.AttributeValueMemberS{Value: "answered"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update item in dynamodb: %w", err)
	}

	return map[string]interface{}{
		"inquiry_id": event.ID,
		"answer":     answer,
		"status":     "success",
	}, nil
}

func main() {
	lambda.Start(handler)
}