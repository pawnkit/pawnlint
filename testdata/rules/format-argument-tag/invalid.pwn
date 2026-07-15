#pragma rational Float

native LogValues(const format[], {Float, _}:...);

main() {
    new Float:ratio;
    new count;
    LogValues("%f %d", count, ratio);
    LogValues("%d", 1.0);
}
