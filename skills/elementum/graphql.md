# Graphql

## graphql - TODO

Execute an arbitrary GraphQL query or mutation against the Elementum API.

The query can be provided as an argument, via stdin, or from a file using -f.

```bash
# Simple query as argument
ei graphql 'query { organization { id name } }'

# Query with variables
ei graphql 'query($id: ID!) { organization { app(id: $id) { name } } }' -v '{"id": "abc-123"}'

# Mutation
ei graphql 'mutation { phoneServiceDelete(id: "abc-123") { id } }'

# From file
ei graphql -f query.graphql

# From stdin
cat query.graphql | ei graphql

# Pretty print output
ei graphql 'query { organization { id } }' --pretty
```

```bash
# TODO Read query from file
ei graphql <query> --file "<value>"

# TODO Pretty print JSON output
ei graphql <query> --pretty

# TODO JSON variables for the query
ei graphql <query> --variables "<value>"
```
