resource "liff_app" "example" {
  description = "Your LIFF App name"
  view = {
    type = "full"
    url  = "https://example.com"
  }
  bot_prompt = "normal"
  scope      = ["profile"]
  feature = {
    qr_code = true
  }
}
