package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type InquiryEvent struct {
	ID string `json:"id"`
}

type InquiryItem struct {
	ID         string `dynamodbav:"id"`
	ReviewText string `dynamodbav:"reviewText"`
}

type BedrockRequestBody struct {
	AnthropicVersion string    `json:"anthropic_version"`
	MaxTokens        int       `json:"max_tokens"`
	Messages         []Message `json:"messages"`
	Temperature      float64   `json:"temperature"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type BedrockResponseBody struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

var (
	dynamoDBClient *dynamodb.Client
	bedrockClient  *bedrockruntime.Client
	inquiryTable   string
	awsRegion      string
)

var categories = map[string]string{
	"質問":     "お客様からの質問や疑問",
	"改善要望":   "サービスや商品の改善に関する要望",
	"ポジティブな感想": "満足度の高い感想や評価",
	"ネガティブな感想": "不満や問題点に関する感想",
	"その他":    "上記に該当しない内容",
}

func init() {
	inquiryTable = os.Getenv("INQUIRY_TABLE_NAME")
	awsRegion = os.Getenv("AWS_REGION")

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion))
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}

	dynamoDBClient = dynamodb.NewFromConfig(cfg)
	bedrockClient = bedrockruntime.NewFromConfig(cfg)
}

func classifyInquiryWithBedrock(ctx context.Context, text string) (string, error) {
	var categoryList []string
	for cat, desc := range categories {
		categoryList = append(categoryList, fmt.Sprintf("- %s: %s", cat, desc))
	}

	prompt := fmt.Sprintf(`以下の問い合わせ内容を分析し、最も適切なカテゴリを1つ選んでください。

カテゴリ:
%s

問い合わせ内容:
%s

指示:
- 上記のカテゴリの中から最も適切なものを1つ選んでください
- カテゴリ名のみを回答してください（説明は不要）
- 判断が困難な場合は「その他」を選んでください

回答:`, strings.Join(categoryList, "\n"), text)

	requestBody, err := json.Marshal(BedrockRequestBody{
		AnthropicVersion: "bedrock-2023-05-31",
		MaxTokens:        50,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal bedrock request body: %w", err)
	}

	modelID := "anthropic.claude-3-haiku-20240307-v1:0"
	out, err := bedrockClient.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     &modelID,
		Body:        requestBody,
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to invoke bedrock model: %w", err)
	}

	var responseBody BedrockResponseBody
	if err := json.Unmarshal(out.Body, &responseBody); err != nil {
		return "", fmt.Errorf("failed to unmarshal bedrock response body: %w", err)
	}

	if len(responseBody.Content) > 0 {
		category := strings.TrimSpace(responseBody.Content[0].Text)
		if _, ok := categories[category]; ok {
			return category, nil
		}
		// 部分一致のチェック
		for cat := range categories {
			if strings.Contains(category, cat) {
				return cat, nil
			}
		}
	}

	return "その他", nil
}

func handler(ctx context.Context, event InquiryEvent) (map[string]interface{}, error) {
	if event.ID == "" {
		return nil, fmt.Errorf("missing inquiry id")
	}

	// DynamoDBから問い合わせ内容を取得
	getItemOutput, err := dynamoDBClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &inquiryTable,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: event.ID},
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

	// Bedrockでカテゴリ分類
	category, err := classifyInquiryWithBedrock(ctx, item.ReviewText)
	if err != nil {
		return nil, fmt.Errorf("failed to classify inquiry: %w", err)
	}

	// DynamoDBにカテゴリを保存
	_, err = dynamoDBClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &inquiryTable,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: event.ID},
		},
		UpdateExpression: aws.String("SET category = :category, #status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":category": &types.AttributeValueMemberS{Value: category},
			":status":   &types.AttributeValueMemberS{Value: "categorized"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update item in dynamodb: %w", err)
	}

	return map[string]interface{}{
		"inquiry_id": event.ID,
		"category":   category,
		"status":     "success",
	},
	nil
}

func main() {
	lambda.Start(handler)
}