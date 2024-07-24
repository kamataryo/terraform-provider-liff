terraform {
  required_providers {
    liff = {
      source = "github.com/kamataryo/liff"
    }
  }
}

provider "liff" {
  channel_id = "1661257543"
  channel_secret = "7fc4899d18befefd4702e416c26a2750"
}

data "liff_app" "sample" {
  liff_id = "1661257543-X62LbpD8"
}

output "test" {
  value = data.liff_app.sample
}
