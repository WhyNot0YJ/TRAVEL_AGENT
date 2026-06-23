export interface FormState {
  departureCity: string;
  destinationCity: string;
  days: number;
  budget: number;
  interests: string;
  transportMode: string;
  pace: string;
}

export function initialFormState(): FormState {
  return {
    departureCity: "上海",
    destinationCity: "杭州",
    days: 3,
    budget: 3000,
    interests: "自然风光，美食",
    transportMode: "train_taxi",
    pace: "balanced",
  };
}
