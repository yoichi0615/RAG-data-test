## 処理フロー
1. 問い合わせをDynamoDBに保存
2. `judge-category` Lambdaで問い合わせ種類を分類（ポジティブ/ネガティブ/質問/改善要望）
3. `create-answer` LambdaがBedrockを使用してRAG検索を実行
4. Claude 3 Haikuが回答を生成
5. 生成された回答をDynamoDBに保存

## アーキテクチャ
![RAG System Architecture](./rag-system-architecture.png)
