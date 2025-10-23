path "home/*" {
    capabilities = ["list", "read"]
}

path "auth/token/lookup-self" {
    capabilities = ["read"]
}

path "auth/token/renew-self" {
    capabilities = ["update"]
}

path "auth/token/revoke-self" {
    capabilities = ["update"]
}

path "sys/capabilities-self" {
    capabilities = ["update"]
}

path "identity/entity/id/{{identity.entity.id}}" {
    capabilities = ["read"]
}

path "identity/entity/name/{{identity.entity.name}}" {
    capabilities = ["read"]
}
