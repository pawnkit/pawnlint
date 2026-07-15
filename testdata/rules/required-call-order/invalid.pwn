native Connect();
native Authenticate();

Test(flag) {
    Authenticate();
    Connect();
    if (flag) {
        Authenticate();
    }
}

Conditional(flag) {
    if (flag) {
        Connect();
    }
    Authenticate();
}

main() {
    Test(1);
    Conditional(1);
}
