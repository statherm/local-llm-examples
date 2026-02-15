MODEL ?= qwen3:4b
SCENARIO ?= all

.PHONY: run score report clean

run:
	go run . -model=$(MODEL) -scenario=$(SCENARIO)

score:
	go run . -score -scenario=$(SCENARIO)

report:
	go run . -report

clean:
	rm -rf results/
