Gem::Specification.new do |spec|
  spec.name          = "chronos-guard-sdk"
  spec.version       = "1.0.0"
  spec.authors       = ["Vamsi Krishna Ommini"]
  spec.email         = ["ovamsikrishna@gmail.com"]
  spec.summary       = "Production-grade Ruby integration bindings for Chronos-Guard microservice proxies"
  spec.files         = Dir["lib/**/*"]
  spec.require_paths = ["lib"]

  spec.add_dependency "grpc", ">= 1.60.0"
  spec.add_dependency "google-protobuf", ">= 3.25.0"
end