resource "liff_app" "exapmple" {
  description = "Your LIFF app name"
  view = {
    type = "full"
    url  = "https://example.com"
  }
  bot_prompt = "aggressive"
  scope      = ["profile"]
  feature = {
    qr_code = true
  }
}
