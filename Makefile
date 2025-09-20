.PHONY: unit-test func-test

# 运行单元测试 (排除test目录下的功能测试)
unit-test:
	@echo "运行单元测试..."
	@go test -gcflags="all=-N -l" -v ./... -run "." -count=1 | grep -v "/test"

# 运行功能测试 (test目录下的测试)
func-test:
	@echo "运行功能测试..."
	@for dir in $$(find . -type d -name "test" | grep -v ".git"); do \
		echo "运行 $$dir 中的功能测试..."; \
		go test -gcflags="all=-N -l" -v $$dir/... -count=1; \
	done