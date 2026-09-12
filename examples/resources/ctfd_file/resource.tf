resource "ctfd_challenge_dynamic" "http" {
  name        = "My Challenge"
  category    = "misc"
  description = "..."
  value       = 500
  decay       = 100
  minimum     = 50
  state       = "visible"
  function    = "logarithmic"

  topics = [
    "Misc"
  ]
  tags = [
    "misc",
    "basic"
  ]
}

resource "ctfd_file" "http_file" {
  challenge_id = ctfd_challenge_dynamic.http.id
  name         = "image.png"
  content_path = "${path.module}/image.png"
  content_hash = filesha256("${path.module}/image.png")
}
