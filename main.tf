// Test terraform configuration

terraform {
  required_providers {
    liff = {
      source = "github.com/kamataryo/liff"
    }
  }
}

provider "liff" {
  channel_id = "1661257543"
  channel_secret = "7fc4899d18befefd4702e416c26a2750" # TODO リポジトリを公開する際には Git Filter で削除する
}

# data "liff_app" "sample" {
#   liff_id = "1661257543-X62LbpD8"
# }

resource "liff_app" "sample" {
  description = "test"
  view = {
    type = "tall"
    url = "https://example.com"
  }
  bot_prompt = "aggressive"
  permanent_link_pattern = "concat"
}

output "create_liff_app" {
  value = liff_app.sample
}
