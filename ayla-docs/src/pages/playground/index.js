import React from "react";
import Layout from "@theme/Layout";
import Playground from "../../components/Playground";
import "../../components/playground.css";

export default function PlaygroundPage() {
  return (
    <Layout
      title="Playground"
      description="Run Ayla Lang code in your browser"
    >
      <Playground />
    </Layout>
  );
}
