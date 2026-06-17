.PHONY: build test examples clean

BIN := kochbahn
YAMLS := $(wildcard examples/*.yaml)
SVGS := $(YAMLS:.yaml=.svg)

build:
	go build -o $(BIN) .

test:
	go test ./...

# Render every examples/*.yaml to a sibling .svg.
examples: $(SVGS)

examples/%.svg: examples/%.yaml build
	./$(BIN) -in $< -out $@

clean:
	rm -f $(BIN) examples/*.svg
