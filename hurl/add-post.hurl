POST http://localhost:3000/add
X-Api-Key: {{QRCODE_API_KEY}}
Content-Type: application/json
{
  "id": 1,
  "name": "Foo",
  "image": "{{BASE64_IMAGE}}"
}
