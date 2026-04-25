IMAGE_TEST ?= notes-test

.PHONY: test

test:
	docker build -f Dockerfile.test -t $(IMAGE_TEST) .
	docker run --rm $(IMAGE_TEST)
