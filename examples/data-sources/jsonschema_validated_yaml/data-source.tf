data "jsonschema_validated_yaml" "example" {
  input_pattern = "./example/**/*.yaml"
}

output "example" {
  value = data.news_validated_yaml.example.values
}