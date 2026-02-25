data "orcasecurity_discovery_results" "myresults" {
  query = jsonencode({
    "type" : "object_set",
    "models" : ["Inventory"],
    "keys" : ["Inventory"],
    "with" : {
      "type" : "operation",
      "values" : [
        { "type" : "str",
          "key" : "Name",
          "values" : ["4d9b3e13-22dd-4861-8b2d-4c8939cb599e"],
          "operator" : "in"
        }
      ],
      "operator" : "and"
    }
  })
  limit          = 2
  start_at_index = 0
}


// you can later access the data returned
output "myresults" {
  value = data.orcasecurity_discovery_results.demo.results[0].name
}
