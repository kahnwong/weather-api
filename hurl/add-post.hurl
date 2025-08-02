POST http://localhost:3000/add
X-Api-Key: {{QRCODE_API_KEY}}
Content-Type: application/json
{
  "id": {{ID}},
  "name": "{{NAME}}",
  "image": "{{BASE64_IMAGE}}"
}
