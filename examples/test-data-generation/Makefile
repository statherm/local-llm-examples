.PHONY: run score report clean

MODEL ?= qwen3:4b

# Generate test data for all schemas with the specified model
run:
	go run . -model $(MODEL)

# Score existing results against schema constraints
score:
	./score.sh

# Generate a comparison report from all results
report:
	go run . -report

# Remove generated results
clean:
	rm -f results/*.json RESULTS.md
