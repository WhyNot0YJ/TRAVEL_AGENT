interface StateViewProps {
  title: string;
  message: string;
}

export default function StateView({ title, message }: StateViewProps) {
  return (
    <section className="state-panel">
      <h2>{title}</h2>
      <p>{message}</p>
    </section>
  );
}
