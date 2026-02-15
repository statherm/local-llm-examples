.PHONY: run-example score report clean tidy

# Run a specific example: make run-example EXAMPLE=structured-extraction MODEL=qwen3:4b
run-example:
	@if [ -z "$(EXAMPLE)" ]; then echo "Usage: make run-example EXAMPLE=<name> [MODEL=<model>]"; exit 1; fi
	cd examples/$(EXAMPLE) && go run . $(if $(MODEL),-model $(MODEL),)

# Score results for an example: make score EXAMPLE=structured-extraction
score:
	@if [ -z "$(EXAMPLE)" ]; then echo "Usage: make score EXAMPLE=<name>"; exit 1; fi
	cd examples/$(EXAMPLE) && go run . -score

# Generate a report for an example: make report EXAMPLE=structured-extraction
report:
	@if [ -z "$(EXAMPLE)" ]; then echo "Usage: make report EXAMPLE=<name>"; exit 1; fi
	cd examples/$(EXAMPLE) && go run . -report

# Remove generated results
clean:
	find examples -name "*.json" -path "*/results/*" -delete

# Tidy Go module
tidy:
	go mod tidy
