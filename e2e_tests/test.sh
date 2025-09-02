#!/bin/bash

servers=("net_http" "gin" "fiber" "yaml_to_code")

for server in "${servers[@]}"; do
  echo "Testing HTTP.$server"
  for compose_file in tests/docker-compose*.yml; do
    filename=$(basename "$compose_file" .yml)
    test_name=${filename#docker-compose-}
    echo "Running HTTP.$server.$test_name"

    SERVER="$server" TESTER_SERVER="http" docker-compose -p "go_template_e2e_testing" -f "$compose_file" up --abort-on-container-exit --exit-code-from tester
    EXIT_CODE=$(docker inspect "tester" --format='{{.State.ExitCode}}')
    SERVER="$server" TESTER_SERVER="http" docker-compose -p "go_template_e2e_testing" -f "$compose_file" down -v
    if [ "$EXIT_CODE" -ne 0 ]; then
      echo "Test HTTP.$server.$test_name failed (exit code $EXIT_CODE)"
      exit 1
    fi
  done
done

echo "Testing GraphQL"
for compose_file in tests/docker-compose*.yml; do
  filename=$(basename "$compose_file" .yml)
  test_name=${filename#docker-compose-}
  echo "Running GraphQL.$test_name"

  SERVER="graphql" TESTER_SERVER="graphql" docker-compose -p "go_template_e2e_testing" -f "$compose_file" up --abort-on-container-exit --exit-code-from tester
  EXIT_CODE=$(docker inspect "tester" --format='{{.State.ExitCode}}')
  SERVER="graphql" TESTER_SERVER="graphql" docker-compose -p "go_template_e2e_testing" -f "$compose_file" down -v
  if [ "$EXIT_CODE" -ne 0 ]; then
    echo "Test GraphQL.$test_name failed (exit code $EXIT_CODE)"
    exit 1
  fi
done

echo "Testing GRPC"
for compose_file in tests/docker-compose*.yml; do
  filename=$(basename "$compose_file" .yml)
  test_name=${filename#docker-compose-}
  echo "Running GRPC.$test_name"

  SERVER="grpc" TESTER_SERVER="grpc" docker-compose -p "go_template_e2e_testing" -f "$compose_file" up --abort-on-container-exit --exit-code-from tester
  EXIT_CODE=$(docker inspect "tester" --format='{{.State.ExitCode}}')
  SERVER="grpc" TESTER_SERVER="grpc" docker-compose -p "go_template_e2e_testing" -f "$compose_file" down -v
  if [ "$EXIT_CODE" -ne 0 ]; then
    echo "Test GRPC.$test_name failed (exit code $EXIT_CODE)"
    exit 1
  fi
done

echo "All tests passed"

