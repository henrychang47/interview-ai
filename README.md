# Interview AI

模擬面試應用，包含 Go 後端、React 前端、PostgreSQL，以及 Gemini / mock AI 模式。

## 需求

- Docker Desktop
- Docker Compose
- Go 1.24
- Node.js 20

## 快速啟動

```powershell
Copy-Item .env.example .env
docker compose up --build
```

服務預設位址：

- 前端：`http://localhost:5173`
- 後端：`http://localhost:8080`

## 資料庫遷移

```powershell
docker compose up -d postgres
docker compose run --rm migrate
```

拉取更新後若有新的 migration，請再執行一次。

確認資料表：

```powershell
docker compose exec postgres psql -U interview_ai -d interview_ai -c "\dt"
```

## 常用驗證

後端健康檢查：

```powershell
curl http://localhost:8080/health
```

預期回應：

```json
{"status":"ok"}
```

前端手動驗證：

1. 開啟 `http://localhost:5173/interviews/new`。
2. 輸入職位資訊、選擇題目數量與語言。
3. 測試麥克風並允許瀏覽器權限。
4. 建立面試後，確認題目準備完成前不能開始。
5. 開始模擬面試後，確認題目朗讀、錄音、回答結束與重新播放功能正常。
6. 完成後確認結果頁可播放回答音檔，並顯示逐字稿與改進建議。

## AI 模式

未設定 `GEMINI_API_KEY` 時，後端會使用 mock 題目產生與 mock 回答分析；題目朗讀會使用瀏覽器 `SpeechSynthesis`。

若要啟用 Gemini，請在 `.env` 設定：

```powershell
GEMINI_API_KEY=
GEMINI_MODEL=gemini-2.5-flash
GEMINI_FALLBACK_MODEL=gemini-2.5-flash-lite
GEMINI_ANSWER_MODEL=gemini-2.5-flash
GEMINI_ANSWER_FALLBACK_MODEL=gemini-2.5-flash-lite
GEMINI_TTS_MODEL=gemini-3.1-flash-tts-preview
GEMINI_TTS_FALLBACK_MODEL=gemini-2.5-flash-preview-tts
GEMINI_TTS_VOICE=Kore
```

請勿提交真實 API key。local secrets 請只放在 `.env`。

## 本機檢查

後端：

```powershell
cd backend
go test ./...
```

前端：

```powershell
cd frontend
npm install
npm run build
```
