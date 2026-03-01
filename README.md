# Kratos Admin Template

## Best Practice
Google AIP(https://google.aip.dev/general):
1. Resource-oriented design
2. Filtering
3. Pagination
4. Field masks
5. Field behavior

## Generate API files
```shell
# Download and update dependencies
make init
# Generate API files (include: pb.go, http, grpc, validate, swagger) by proto file
make api
```
