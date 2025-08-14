% COMPLYCTL PLAN-EDIT(2)
% Hannah Braswell <hbraswel@redhat.com>
% August 2025

# complyctl plan

## complyctl plan <framework-id> --dry-run

The `--dry-run` flag will print the default values of the framework to standard out.

The output will inform the user of the default values from the framework that will be included in the assessment plan.

```yaml

frameworkId: anssi_bp28_minimal
includeControls:
- controlId: r80
  controlTitle: Minimization of Network Services
  includeRules:
  - "*"
  selectParameters:
  - name: var_selinux_state
    value: xyz
  - name: var_password_pam_minlen
    value: xyz
```

## Scoping your Assessment Plan

### `complyctl plan <framework-id> --dry-run --out config.yml`

To configure your assessment plan, the `config.yml` fields need to be updated.

**Controls:** The `controlId`, `controlTitle`, `includeRules`, and `selectParameters` yaml keys are grouped for each distinct control. To exclude a control from the assessment plan, delete the entire subset of yaml keys associated with the control-id. After deletion, the controls will be excluded from the assessment plan and the activities associated with those out of scope controls will be marked as "skipped."

**Rules:** The `globalExcludeRules` yaml key can be placed at the end of the `config.yml` to globally exclude a rule from all controls in the assessment plan. The `includeRules` field defaults to a global wildcard (*) that will select all rules for the particular `controlId`. The `excludeRules` yaml key has to be manually added under each `controlId` to specifically exclude a rule within a control. The intersection of `excludeRules` and `globalExcludeRules`

**Parameters:** The `selectParameters` yaml key has two second-level yaml keys for the name and value of each parameter associated with the includedControls of the assessment plan. The `complyctl info <framework-id> --parameter <parameter-id>` command will print the alternative selections for the parameter-id. To select parameters as attributes of your assessment plan, use the alternative selections from the `info` command to change the `value` of the parameter. Then, the assessment plan properties will reflect the alternative selection value that was indicated in the `config.yml`.

## complyctl plan <framework-id> --scope-config config.yml


To configure the parameters of your assessment plan