# Provider form layout design

## Goal

Make the provider setup screen readable and safe to use during real transactions. The collateral and offer controls must preserve label/input relationships, explain irreversible or delayed actions, and remain aligned from wide desktop screens down to mobile.

## Existing failure

`Offers` places labels, inputs, a checkbox, and the submit button directly into a three-column CSS grid. Each element becomes an independent grid item, so labels and their controls can land in different rows or columns. `ProviderConsole` similarly mixes the bonded balance, deposit form, and withdrawal action in one wrapping flex row without enough hierarchy or transaction guidance.

## Approved layout

The provider operations section will contain two bordered cards using the existing Myference dashboard tokens.

### Collateral card

- Lead with a compact explanation that collateral backs provider behavior and remains owned by the provider unless slashed.
- Show the confirmed bonded balance as a MON value with the exact wei value available as supporting data.
- Group the amount label, input, `MON` unit, and minimum-bond helper text as one field.
- Keep the primary `Deposit collateral` action beside the secondary bond-exit action on wide screens and stack them on narrow screens.
- Explain that deposits are immediate after chain confirmation and withdrawals use the configured exit delay.

### Offer card

- Explain that publishing creates a new immutable on-chain price version; existing sessions keep their pinned version.
- Divide the form into two semantic fieldsets: model identity and pricing.
- Use a two-column field grid on desktop and one column on narrow screens.
- Every field uses a wrapper containing its label, control, unit where relevant, and concise helper text.
- Keep contract-facing price inputs as non-negative integer wei to avoid lossy decimal conversion.
- Explain input/output pricing as wei per one million tokens and compute pricing as wei per second.
- Present temporary workspace as a full-width checkbox row with a warning that it is only for isolated CLI coding agents.
- Place the submit action in a dedicated footer with an immutability reminder.

## Accessibility and responsive behavior

- Preserve explicit `htmlFor`/`id` relationships and native form controls.
- Use `fieldset` and `legend` for the two offer groups.
- Use `inputMode="numeric"` for wei values and `inputMode="decimal"` for MON.
- Helper text is connected to inputs with `aria-describedby`.
- Focus styling continues to use the global dashboard focus treatment.
- At 760 px and below, all field grids and action rows become single-column without reordering labels away from controls.

## Testing

- Component tests assert the two semantic cards and fieldsets render.
- Tests assert every relevant input has its explanatory text through `aria-describedby`.
- Existing provider transaction tests continue to cover deposit, bond exit, and offer publication behavior.
- The full web test suite, lint, and production build must pass.
- Browser verification covers the provider screen at desktop and mobile widths before deployment.

## Scope

This change fixes provider-form structure, copy, accessibility, and responsive styling. It does not change contract calls, pricing units, transaction rules, or provider accounting.
