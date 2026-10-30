import json
from pathlib import Path


def test_cookbook_sample_files_exist_and_valid():
    samples_dir = Path("data/raw/samples")
    assert samples_dir.exists(), "data/raw/samples directory must exist"

    expected_files = [
        "sample_underlying_identity.json",
        "sample_underlying_daily.json",
        "sample_underlying_ticks.json",
        "sample_option_search.json",
        "sample_option_daily.json",
        "sample_option_ticks.json",
    ]

    for filename in expected_files:
        filepath = samples_dir / filename
        assert filepath.exists(), f"Sample file {filename} must exist"
        assert filepath.stat().st_size > 0, f"Sample file {filename} must not be empty"

        with open(filepath, encoding="utf-8") as f:
            data = json.load(f)
            assert isinstance(data, dict), f"Sample file {filename} must be valid JSON dict"


def test_sample_ticks_and_daily_structures():
    samples_dir = Path("data/raw/samples")

    # Verify underlying ticks
    with open(samples_dir / "sample_underlying_ticks.json", encoding="utf-8") as f:
        ticks_data = json.load(f)
        assert "tradeHistory" in ticks_data
        assert len(ticks_data["tradeHistory"]) > 0
        first_tick = ticks_data["tradeHistory"][0]
        assert "pTran" in first_tick
        assert "qTitTran" in first_tick
        assert "hEven" in first_tick

    # Verify option search
    with open(samples_dir / "sample_option_search.json", encoding="utf-8") as f:
        search_data = json.load(f)
        assert "instrumentSearch" in search_data
        assert len(search_data["instrumentSearch"]) > 0
        first_option = search_data["instrumentSearch"][0]
        assert "insCode" in first_option
        assert "lVal18AFC" in first_option
