.PHONY: run score report clean

MODEL ?= qwen3:4b

run:
	go run . -model $(MODEL)

score:
	go run . -score

report:
	go run . -report

clean:
	rm -f results/*.json
