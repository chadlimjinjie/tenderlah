"use client"

import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card"
import { Field, FieldLabel, FieldError } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { authErrorMessage } from "@/features/auth/hooks"
import { useCompleteOnboarding, useOnboardingStatus } from "@/features/onboarding/hooks"
import { LogoPicker } from "@/features/onboarding/components/logo-picker"
import {
    COMPANY_SIZES,
    CONTRACT_SIZES,
} from "@/features/onboarding/components/onboarding-form"
import type { OnboardingRequestAccountType, UserOnboarding } from "@/lib/api/generated/model"
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select"

const BusinessProfileSchema = z.object({
    orgName: z.string().min(1, "Organisation name is required."),
    businessName: z.string().optional(),
    businessDescription: z.string().optional(),
    industry: z.string().min(1, "Industry is required."),
    geographicCoverage: z.string().optional(),
})

type BusinessProfileValues = z.infer<typeof BusinessProfileSchema>

export function BusinessProfileCard() {
    const { data: onboarding } = useOnboardingStatus()
    if (!onboarding) return null
    return <BusinessProfileForm onboarding={onboarding} />
}

// Split out so companySize/preferredContractSize's useState can read their initial
// value directly from `onboarding` on first render. Initializing them to "" and
// only setting the real value later (via an effect, once the parent's query
// resolves) makes Radix's Select controlled `value` jump from "" to a real value
// post-mount — when that happens before the dropdown has ever been opened (so its
// items, and the hidden native <select>'s <option>s, haven't mounted yet), the
// browser rejects the unmatched value on Radix's internal bubble <select> and its
// change handler calls onValueChange("") right back, silently reverting it.
// Mounting this component only once `onboarding` exists sidesteps that transition
// entirely — the value is correct from the very first render.
function BusinessProfileForm({ onboarding }: { onboarding: UserOnboarding }) {
    const completeOnboarding = useCompleteOnboarding()

    const {
        register,
        handleSubmit,
        formState: { errors },
    } = useForm<BusinessProfileValues>({
        resolver: zodResolver(BusinessProfileSchema),
        defaultValues: {
            orgName: onboarding.orgName,
            businessName: onboarding.businessName ?? "",
            businessDescription: onboarding.businessDescription ?? "",
            industry: onboarding.industry,
            geographicCoverage: onboarding.geographicCoverage ?? "",
        },
    })

    const [companySize, setCompanySize] = useState(onboarding.companySize ?? "")
    const [preferredContractSize, setPreferredContractSize] = useState(
        onboarding.preferredContractSize ?? ""
    )

    function onSubmit(values: BusinessProfileValues) {
        completeOnboarding.mutate(
            {
                accountType: onboarding.accountType as OnboardingRequestAccountType,
                orgName: values.orgName,
                businessName: values.businessName?.trim() ? values.businessName.trim() : undefined,
                businessDescription: values.businessDescription?.trim()
                    ? values.businessDescription.trim()
                    : undefined,
                industry: values.industry,
                companySize: companySize || undefined,
                geographicCoverage: values.geographicCoverage?.trim() || undefined,
                preferredContractSize: preferredContractSize || undefined,
            },
            {
                onSuccess: () => toast.success("Business profile updated."),
                onError: (err) => toast.error(authErrorMessage(err)),
            }
        )
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle>Business profile</CardTitle>
                <CardDescription>Details collected during onboarding.</CardDescription>
            </CardHeader>
            <CardContent>
                <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
                    <Field>
                        <FieldLabel>Business logo</FieldLabel>
                        <LogoPicker logoUrl={onboarding.logoUrl} />
                    </Field>

                    <Field>
                        <FieldLabel htmlFor="orgName">Organisation name</FieldLabel>
                        <Input id="orgName" {...register("orgName")} />
                        {errors.orgName && <FieldError>{errors.orgName.message}</FieldError>}
                    </Field>

                    <Field>
                        <FieldLabel htmlFor="businessName">Business name (optional)</FieldLabel>
                        <Input id="businessName" {...register("businessName")} />
                    </Field>

                    <Field>
                        <FieldLabel htmlFor="businessDescription">
                            Business description (optional)
                        </FieldLabel>
                        <Textarea id="businessDescription" {...register("businessDescription")} />
                    </Field>

                    <Field>
                        <FieldLabel htmlFor="industry">Industry</FieldLabel>
                        <Input id="industry" {...register("industry")} />
                        {errors.industry && <FieldError>{errors.industry.message}</FieldError>}
                    </Field>

                    <Field>
                        <FieldLabel htmlFor="companySize">Company size (optional)</FieldLabel>
                        <Select value={companySize} onValueChange={setCompanySize}>
                            <SelectTrigger id="companySize" className="w-full">
                                <SelectValue placeholder="Select company size">
                                    {companySize ? `${companySize} employees` : undefined}
                                </SelectValue>
                            </SelectTrigger>
                            <SelectContent>
                                {COMPANY_SIZES.map((option) => (
                                    <SelectItem key={option} value={option}>
                                        {option} employees
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </Field>

                    <Field>
                        <FieldLabel htmlFor="geographicCoverage">
                            Geographic coverage (optional)
                        </FieldLabel>
                        <Input
                            id="geographicCoverage"
                            placeholder="e.g. Singapore-wide"
                            {...register("geographicCoverage")}
                        />
                    </Field>

                    <Field>
                        <FieldLabel htmlFor="preferredContractSize">
                            Preferred contract size (optional)
                        </FieldLabel>
                        <Select value={preferredContractSize} onValueChange={setPreferredContractSize}>
                            <SelectTrigger id="preferredContractSize" className="w-full">
                                <SelectValue placeholder="Select preferred contract size">
                                    {preferredContractSize || undefined}
                                </SelectValue>
                            </SelectTrigger>
                            <SelectContent>
                                {CONTRACT_SIZES.map((option) => (
                                    <SelectItem key={option} value={option}>
                                        {option}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </Field>

                    <div>
                        <Button type="submit" disabled={completeOnboarding.isPending}>
                            {completeOnboarding.isPending ? "Saving…" : "Save changes"}
                        </Button>
                    </div>
                </form>
            </CardContent>
        </Card>
    )
}
