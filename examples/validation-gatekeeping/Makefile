.PHONY: run score report clean

MODEL ?= qwen3:4b

run:
	go run . -model $(MODEL)

run-prompts:
	go run . -model $(MODEL) -scenario prompts

run-pii:
	go run . -model $(MODEL) -scenario pii

score:
	go run . -score

report:
	go run . -report

clean:
	rm -f results/*.json
