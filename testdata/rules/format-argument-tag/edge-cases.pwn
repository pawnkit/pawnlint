#pragma rational Float

native LogValues(const format[], {Float, _}:...);

main() {
    new Float:value;
    new dynamicFormat[8] = "%d";
    LogValues(dynamicFormat, value);
    LogValues("%s", value);
}
