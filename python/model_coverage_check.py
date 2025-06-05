from pathlib import Path
import ast

# === CONFIGURATION ===
MODELS_DIR = Path("../api/models/")
FLOWS_DIR = Path("./flows/crud/")

# === MANUAL OVERRIDES ===
# ModelName: filename or reason (e.g. "Ignore", "partial_flow.py", or "" if unknown yet)
MANUAL_MAPPING = {}


def snake_to_pascal(s):
    return "".join(word.capitalize() for word in s.split("_"))


def get_model_names(models_dir):
    model_names = set()

    for file in models_dir.glob("*.py"):
        with open(file) as f:
            tree = ast.parse(f.read())

        for node in ast.walk(tree):
            if isinstance(node, ast.ClassDef):
                for base in node.bases:
                    if (isinstance(base, ast.Name) and base.id == "Model") or (
                        isinstance(base, ast.Attribute) and base.attr == "Model"
                    ):
                        model_names.add(node.name)

    return model_names


def get_flow_targets(flows_dir):
    return {snake_to_pascal(file.stem) for file in flows_dir.glob("*.py")}


def main():
    model_names = get_model_names(MODELS_DIR)
    flow_targets = get_flow_targets(FLOWS_DIR)

    print("=== Model Flow Coverage ===\n")
    for model in sorted(model_names):
        if model in MANUAL_MAPPING:
            print(
                f"⏭️  {model} — manually handled: {MANUAL_MAPPING[model] or '(pending)'}"
            )
            continue
        if model in flow_targets:
            print(f"✅ {model}")
        else:
            print(f"❌ {model} — no corresponding flow")

    # Infer manual flow coverage from values in MANUAL_MAPPING that end in ".py"
    manually_covered_flows = {
        Path(val).stem
        for val in MANUAL_MAPPING.values()
        if isinstance(val, str) and val.endswith(".py")
    }

    extra_flows = (
        flow_targets
        - model_names
        - set(MANUAL_MAPPING.keys())
        - {snake_to_pascal(name) for name in manually_covered_flows}
    )

    if extra_flows:
        print("\n⚠️ Flow files with no matching model:")
        for flow in sorted(extra_flows):
            print(f"🟡 {flow}")

    print("\n=== Manual Mapping Summary ===")
    for model, val in MANUAL_MAPPING.items():
        status = "✅" if val else "❌"
        print(f"{status} {model}: {val or '(empty — needs attention)'}")


if __name__ == "__main__":
    main()
