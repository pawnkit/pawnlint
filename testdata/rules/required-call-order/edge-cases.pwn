native Connect();
native Authenticate();

Test(flag) {
    if (flag) {
        Connect();
        Authenticate();
    }
}

main() {
    Test(1);
}
