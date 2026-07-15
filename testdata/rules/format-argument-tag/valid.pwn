#pragma rational Float

native LogValues(const format[], {Float, Custom, _}:...);

main() {
    new Float:ratio;
    new count;
    new Custom:identifier;
    LogValues("%f %d %x", ratio, count, identifier);
    LogValues("%f", 1.0);
}
