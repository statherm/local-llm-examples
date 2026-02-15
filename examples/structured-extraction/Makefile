MODEL ?= qwen3:4b
SCENARIO ?= all

.PHONY: run score clean

run:
	go run . -model=$(MODEL) -scenario=$(SCENARIO)

score:
	@echo "Scoring requires result files. Run 'make run' first to generate results."
	@echo "Then use: ./score.sh <results-dir> expected/<scenario>"

clean:
	rm -rf results/
