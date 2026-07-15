native Connect();
native Authenticate();
native Query();

main() {
    Connect();
    Authenticate();
    Query();
}

ValidBranches(flag) {
    if (flag) {
        Connect();
    } else {
        Connect();
    }
    Authenticate();
}
